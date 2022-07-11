package socket

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type router_conn_obj struct {
	raddr *net.TCPAddr
	conn  *net.TCPConn

	recovering bool
}

type router_conn_obj_ptr *router_conn_obj

type conc_router_conns struct {
	conns map[string]*router_conn_obj
	sync.RWMutex
}

var router_conns *conc_router_conns

const sock_id = "SOCK_main"
const CONN_GROUP = "AZ1_main"
const BUFFER_SIZE = 1000000 //1 mb
var HEARTBEAT_INTERVAL = 10 * time.Second
var HEARTBEAT_MAX_DELAY = HEARTBEAT_INTERVAL
var RETRY_INTERVAL = 1 * time.Second
var MAX_RETRIES = 3

var router_addr []string = []string{"127.0.0.1:10000"}
var socket_writer_pointer unsafe.Pointer

var socket_logger *log.Logger
var websocket_logger *log.Logger

func InitializeSocketConn() {

	socket_logs_file, err := os.OpenFile("socket_logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln(err)
	}

	socket_logger = log.New(socket_logs_file, "", log.LstdFlags)

	websocket_logs_file, err := os.OpenFile("websocket_logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln(err)
	}

	websocket_logger = log.New(websocket_logs_file, "", log.LstdFlags)

	router_conns = &conc_router_conns{
		conns: make(map[string]*router_conn_obj),
	}

	for i := 0; i < len(router_addr); i++ {
		go connect_new_socket(router_addr[i], "nil")
	}

}

func CloseSocketConn() {

	router_conns.Lock()

	for _, v := range router_conns.conns {
		v.conn.Close()
	}

	router_conns.Unlock()

}

func connect_new_socket(addr string, old_conn_id string) {

	raddr, _ := net.ResolveTCPAddr("tcp", addr)
	conn, err := net.DialTCP("tcp", nil, raddr)

	if err != nil {
		handle_panic_err(conn, err)
	}

	defer conn.Close()

	_, err = conn.Write(package_op(2, []string{sock_id, CONN_GROUP, old_conn_id}, nil))
	if err != nil {
		handle_panic_err(conn, err)
	}

	res_b := make([]byte, 6)
	_, err = conn.Read(res_b)
	if err != nil {
		handle_panic_err(conn, err)
	}

	if res_b[0] != 2 {
		handle_panic_err(conn, err)
	}

	if string(res_b[5]) == "0" {
		fmt.Println("Recovering from new socket")
		recover_socket(&router_conn_obj{raddr: raddr, recovering: false})
	}

	router_conn := &router_conn_obj{
		raddr:      raddr,
		conn:       conn,
		recovering: false,
	}

	read_tcp(router_conn)

}

func recover_socket(router_conn *router_conn_obj) {

	router_conns.Lock()

	if router_conn.recovering {
		router_conns.Unlock()
		return
	}

	if router_conns.conns[router_conn.raddr.IP.String()] != nil {
		router_conns.conns[router_conn.raddr.IP.String()].recovering = true
	}

	router_conns.Unlock()

	var conn *net.TCPConn
	var err error

	i := 0
	for {

		if i == 3 {
			log.Panicln("ERROR:", "COULD NOT RECOVER SOCKET")
		}
		i++

		fmt.Println(router_conn.raddr.IP.String()+" closed, recovering...", i)

		conn, err = net.DialTCP("tcp", nil, router_conn.raddr)
		if err != nil {
			log_err("recovering " + sock_id + " failed: " + err.Error())
			continue
		}

		_, err = conn.Write(package_op(3, []string{sock_id}, nil))
		if err != nil {
			log_err("recovering " + sock_id + " failed: " + err.Error())
			continue
		}

		res_b := make([]byte, 6)
		_, err = conn.Read(res_b)
		if err != nil {
			log_err("recovering " + sock_id + " failed: " + err.Error())
			continue
		}

		if res_b[0] != 3 {
			log_err("recovering " + sock_id + " failed: " + "unexpected op: " + string(res_b[0]))
			continue
		}

		if string(res_b[5]) == "0" {
			connect_new_socket(router_conn.raddr.IP.String()+":"+strconv.Itoa(router_conn.raddr.Port), "nil")
			continue
		}

		break
	}

	defer conn.Close()

	router_conn = &router_conn_obj{
		raddr:      router_conn.raddr,
		conn:       conn,
		recovering: false,
	}

	read_tcp(router_conn)
}

func read_tcp(router_conn *router_conn_obj) {

	var received uint32

	heartbeat_chan := make(chan []byte, 3)
	timeout_chan := make(chan byte)

	defer func() {
		close(heartbeat_chan)
	}()

	go heartbeat_check(router_conn, heartbeat_chan, timeout_chan)
	go heartbeat_worker(router_conn, &received, timeout_chan)

	router_conns.Lock()
	router_conns.conns[router_conn.raddr.IP.String()] = router_conn
	router_conns.Unlock()

	atomic.StorePointer(&socket_writer_pointer, unsafe.Pointer(router_conn))

	fmt.Println("Connection with sock_id " + sock_id + " is ready...")

	reader := bufio.NewReaderSize(router_conn.conn, BUFFER_SIZE)

	var (
		err    error
		header []byte = make([]byte, 5)
		size   uint32
		op     byte
		b      []byte
	)
	for {

		_, err = io.ReadFull(reader, header)
		if err != nil {
			handle_conn_err(router_conn, err)
			break
		}

		op = header[0]

		size = binary.BigEndian.Uint32(header[1:])
		b = nil

		if size > 0 {
			b = make([]byte, size)
			_, err = io.ReadFull(reader, b)
			if err != nil {
				handle_conn_err(router_conn, err)
				break
			}
		}

		//fmt.Println("SOCK RECV:", header, b)

		if op == 1 {
			heartbeat_chan <- b
			atomic.StoreUint32(&received, 0)
		} else {
			socket_op(router_conn, op, b, &received)
		}
	}

}

func socket_write(op byte, payload []byte, params ...string) error {

	var err error

	writer := *(router_conn_obj_ptr)(atomic.LoadPointer(&socket_writer_pointer))

	if writer.conn == nil {
		return errors.New("unable to write to nil conn")
	}

	fmt.Println("SOCKET WRITING: ", op, string(payload), params)

	for i := 0; i < MAX_RETRIES; i++ {
		_, err = writer.conn.Write(package_op(op, params, payload))
		if err == nil {
			break
		}
		if !err.(*net.OpError).Temporary() {
			break
		}
	}

	if err != nil {
		handle_conn_err(&writer, errors.New("write faliure: "+err.Error()))
	}

	return err
}

func heartbeat_worker(router_conn *router_conn_obj, received *uint32, timeout_chan chan byte) {

	var (
		heartbeat_ver                byte  = 1
		expected_heartbeat_unix_nano int64 = 0
	)

	write_b := make([]byte, 18)
	copy(write_b, []byte{1, 0, 0, 0, 13})

	for {

		expected_heartbeat_unix_nano = time.Now().UnixNano() + int64(HEARTBEAT_INTERVAL)

		write_b[5] = heartbeat_ver
		binary.BigEndian.PutUint64(write_b[6:14], uint64(expected_heartbeat_unix_nano))
		binary.BigEndian.PutUint32(write_b[14:18], atomic.LoadUint32(received))

		//fmt.Println("Heartbeat write:", write_b)

		_, err := router_conn.conn.Write(write_b)
		if err != nil {
			handle_conn_err(router_conn, err)
			fmt.Println("SOCKET HEARTBEAT WORKER QUIT")
			return
		}

		go func(ver byte) {
			time.Sleep(HEARTBEAT_MAX_DELAY)
			timeout_chan <- ver
		}(heartbeat_ver)

		time.Sleep(time.Duration(expected_heartbeat_unix_nano - time.Now().UnixNano()))
		heartbeat_ver++

	}

}

func heartbeat_check(router_conn *router_conn_obj, heartbeat_chan chan []byte, timeout_chan chan byte) {

	var (
		heartbeat_ver byte = 1
		ver           byte
		b             []byte
		ok            bool
	)
	for {
		select {
		case b, ok = <-heartbeat_chan:
			//fmt.Println("Heartbeat:", ver)
			if !ok {
				return
			}

			if b[0] != heartbeat_ver {
				handle_conn_err(router_conn, errors.New("heartbeat version mismatch. Incoming ver: "+fmt.Sprint(ver)+". Heartbeat ver: "+fmt.Sprint(heartbeat_ver)))
				return
			}

			heartbeat_ver++
		case ver = <-timeout_chan:
			if ver == heartbeat_ver {
				handle_conn_err(router_conn, errors.New("heartbeat timeout. Timeout ver: "+fmt.Sprint(ver)+". Heartbeat ver: "+fmt.Sprint(heartbeat_ver)))
				return
			}
		}
	}

}

func handle_panic_err(conn *net.TCPConn, err error) {
	if conn != nil {
		conn.Close()
	}
	log.Panicln("ERROR", err.Error())
}

func handle_conn_err(router_conn *router_conn_obj, err error) {
	if router_conn.conn != nil {
		router_conn.conn.Close()
	}
	log_err("Conn err: " + err.Error())
	recover_socket(router_conn)
}

func log_err(err string) {
	socket_logger.Println(err)
}
