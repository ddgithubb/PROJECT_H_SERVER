package services

import (
	"PROJECT_H_server/config"
	"PROJECT_H_server/errors"
	"PROJECT_H_server/helpers"
	"bytes"
	"encoding/base64"
	Errors "errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/pebbe/zmq4"
)

// Stream starts and maintains websocket connection
func Stream(ws *websocket.Conn) {

	var heartbeat = 0
	var version = 0
	fmt.Println("opened")
	defer ws.Close()
	forceClose := make(chan bool)
	close := false
	go func() {
		close = <-forceClose
	}()

	userID := ws.Locals("userid").(string)
	username := ws.Locals("username").(string)

	pub, err := zmq4.NewSocket(zmq4.PUB)
	if err != nil {
		errors.HandleWebsocketError(ws, "websocket_pub", err.Error())
	}
	defer pub.Close()
	err = pub.Connect("tcp://127.0.0.1:" + config.Config.PubPort)
	if err != nil {
		errors.HandleWebsocketError(ws, "websocket_connect", err.Error())
	}

	go func() {
		sub, err := zmq4.NewSocket(zmq4.SUB)
		if err != nil {
			errors.HandleWebsocketError(ws, "websocket_sub", err.Error())
		}
		defer sub.Close()
		err = sub.Connect("tcp://127.0.0.1:" + config.Config.SubPort)
		if err != nil {
			errors.HandleWebsocketError(ws, "websocket_sub_connect", err.Error())
		}
		err = sub.SetSubscribe(userID)
		if err != nil {
			errors.HandleWebsocketError(ws, "websocket_sub_set", err.Error())
		}

		var packets []string
		for {
			if packets, err = sub.RecvMessage(0); err != nil {
				errors.HandleWebsocketError(ws, "websocket_sub_recv", err.Error())
				break
			}
			if close {
				break
			}
			if err = ws.WriteMessage(websocket.TextMessage, []byte(packets[1])); err != nil {
				errors.HandleWebsocketError(ws, "websocket_write", err.Error())
				break
			}
			if version < 999 {
				version++
			} else {
				version = 0
			}
		}
	}()

	go func() {
		for {
			if heartbeat >= 5 {
				break
			}
			if close {
				break
			}
			if err = ws.WriteMessage(websocket.TextMessage, []byte("PING"+fmt.Sprint(version))); err != nil {
				errors.HandleWebsocketError(ws, "websocket_write_PING", err.Error())
				break
			}
			heartbeat++
			time.Sleep(time.Second * 50) //50 seconds
		}
	}()

	var (
		mt       int
		msg      []byte
		req      string
		reqChunk []string
		payload  []string
		send     bool
		data     *bytes.Buffer
	)
	for {
		if err = ws.SetReadDeadline(time.Now().Add(time.Second * 190)); err != nil {
			errors.HandleWebsocketError(ws, "websocket_read_deadline", err.Error())
			break
		}
		if mt, msg, err = ws.ReadMessage(); err != nil {
			if err != websocket.ErrCloseSent && !websocket.IsCloseError(err, 1000) && !strings.Contains(err.Error(), "i/o timeout") {
				errors.HandleWebsocketError(ws, "websocket_read", err.Error())
			}
			break
		}
		if mt == websocket.BinaryMessage {
			errors.HandleWebsocketError(ws, "websocket_read", "binary message")
			break
		}

		req = string(msg)

		fmt.Println(req)

		if req == "PONG" {
			heartbeat = 0
			continue
		}

		reqChunk = strings.Split(req, "|") //DOPN"T USE SPLIT, SIMPLY USE A STRAIGHT STRING AND INDEX
		payload = []string{reqChunk[0]}
		send = true

		// exists, err = helpers.CheckUser(reqChunk[1])
		// if err != nil {
		// 	if err == gocql.ErrNotFound {
		// 		continue
		// 	}
		// 	break
		// }
		// if !exists {
		// 	continue
		// }

		switch reqChunk[0] {
		case "request":
			payload = append(payload, userID, username, reqChunk[2])
		case "unrequest":
			payload = append(payload, userID)
		case "accept":
			payload = append(payload, userID)
		case "unfriend":
			payload = append(payload, userID)
		case "get-audio":

			send = false
			data, err = helpers.GetAudio(reqChunk[1], reqChunk[2], reqChunk[3])

			if err != nil {
				break
			}

			fmt.Println(data.Len())

			if err = ws.WriteMessage(websocket.TextMessage, []byte("byte"+base64.StdEncoding.EncodeToString(data.Bytes()))); err != nil {
				err = Errors.New("websocket_write: " + err.Error())
				break
			}
		default:
			err = Errors.New("type error")
		}

		if err != nil {
			errors.HandleWebsocketError(ws, "websocket_type", err.Error())
			break
		}

		if send {
			pub.SendMessage(reqChunk[1], strings.Join(payload, "|"))
		}
	}

	forceClose <- true
	fmt.Println(userID, "closed")

}
