package socket

// import (
// 	"PROJECT_H_server/config"
// 	"PROJECT_H_server/errors"
// 	"PROJECT_H_server/helpers"
// 	Errors "errors"
// 	"fmt"
// 	"strings"
// 	"time"

// 	"github.com/gofiber/websocket/v2"
// )

// // Stream starts and maintains websocket connection
// func Stream(ws *websocket.Conn) {

// 	version := 0
// 	defer ws.Close()
// 	forceClose := make(chan bool)
// 	close := false
// 	go func() {
// 		close = <-forceClose
// 	}()

// 	userID := ws.Locals("userid").(string)
// 	username := ws.Locals("username").(string)
// 	//sessionID := ws.Locals("sessionid").(string)

// 	pub, err := zmq4.NewSocket(zmq4.PUB)
// 	if err != nil {
// 		errors.HandleWebsocketError(ws, "websocket_pub", err.Error())
// 	}
// 	defer pub.Close()
// 	err = pub.Connect("tcp://127.0.0.1:" + config.Config.PubPort)
// 	if err != nil {
// 		errors.HandleWebsocketError(ws, "websocket_connect", err.Error())
// 	}

// 	go func() {
// 		sub, err := zmq4.NewSocket(zmq4.SUB)
// 		if err != nil {
// 			errors.HandleWebsocketError(ws, "websocket_sub", err.Error())
// 		}
// 		defer sub.Close() //This should be triggered on forceClose
// 		err = sub.Connect("tcp://127.0.0.1:" + config.Config.SubPort)
// 		if err != nil {
// 			errors.HandleWebsocketError(ws, "websocket_sub_connect", err.Error())
// 		}
// 		err = sub.SetSubscribe(userID)
// 		if err != nil {
// 			errors.HandleWebsocketError(ws, "websocket_sub_set", err.Error())
// 		}

// 		if err = ws.WriteMessage(websocket.TextMessage, []byte("START")); err != nil {
// 			errors.HandleWebsocketError(ws, "websocket_write_START", err.Error())
// 			forceClose <- true
// 			return
// 		}

// 		var packets []string
// 		for {
// 			if packets, err = sub.RecvMessage(0); err != nil {
// 				errors.HandleWebsocketError(ws, "websocket_sub_recv", err.Error())
// 				break
// 			}
// 			if close { //This is not a reliable garbage collection source at all
// 				break
// 			}
// 			if err = ws.WriteMessage(websocket.TextMessage, []byte(packets[1])); err != nil {
// 				errors.HandleWebsocketError(ws, "websocket_write", err.Error())
// 				break
// 			}
// 			if version < 999 {
// 				version++
// 			} else {
// 				version = 0
// 			}
// 		}
// 	}()

// 	var (
// 		mt                int
// 		msg               []byte
// 		req               string
// 		reqID             string
// 		tempChunks        []string
// 		reqChunk          []string
// 		reqText           string
// 		payloadRecv       []string
// 		payloadSend       []string
// 		expectedArgLength int
// 		send              bool
// 	)
// 	for {
// 		if err = ws.SetReadDeadline(time.Now().Add(time.Second * 300)); err != nil {
// 			errors.HandleWebsocketError(ws, "websocket_read_deadline", err.Error())
// 			break
// 		}
// 		if mt, msg, err = ws.ReadMessage(); err != nil {
// 			fmt.Println("WEBSOCKET READ ERR with error : ", err)
// 			//err != websocket.ErrCloseSent && !websocket.IsCloseError(err, 1000)
// 			//errors.HandleWebsocketError(ws, "websocket_read", err.Error())
// 			break
// 		}
// 		if mt == websocket.BinaryMessage {
// 			errors.HandleWebsocketError(ws, "websocket_read", "binary message")
// 			break
// 		}

// 		req = string(msg)

// 		fmt.Println(req)

// 		if req == "PING" {
// 			if err = ws.WriteMessage(websocket.TextMessage, []byte("PONG"+fmt.Sprint(version))); err != nil {
// 				errors.HandleWebsocketError(ws, "websocket_write_PONG", err.Error())
// 				break
// 			}
// 			continue
// 		}

// 		tempChunks = strings.SplitN(req, "|text:", 2)
// 		reqChunk = strings.Split(tempChunks[0], "|")

// 		if len(reqChunk) < 2 {
// 			errors.HandleWebsocketError(ws, "websocket_empty", err.Error())
// 			break
// 		}

// 		if len(tempChunks) > 1 {
// 			reqText = tempChunks[1]
// 		}
// 		fmt.Println(reqText)

// 		payloadSend = []string{reqChunk[0], userID}
// 		payloadRecv = reqChunk[2:]
// 		reqID = reqChunk[1]
// 		send = true

// 		switch reqChunk[0] {
// 		case "request":
// 			expectedArgLength = 1
// 			if payloadRecv[0] != username {
// 				err = Errors.New("Username doesn't match")
// 			}
// 		case "unrequest":
// 			expectedArgLength = 0
// 		case "accept":
// 			expectedArgLength = 1
// 		case "unfriend":
// 			expectedArgLength = 0
// 		case "send-message":
// 			expectedArgLength = 8
// 			reqText = "undefined"
// 		case "send-reply":
// 			expectedArgLength = 3
// 			err = helpers.UpdateReply(reqChunk[2], reqChunk[3], reqChunk[4])

// 			if err != nil {
// 				break
// 			}
// 		default:
// 			err = Errors.New("type error")
// 		}

// 		if err != nil {
// 			errors.HandleWebsocketError(ws, "websocket_type", err.Error())
// 			break
// 		}

// 		if len(payloadRecv) != expectedArgLength {
// 			err = Errors.New("invalid_args")
// 			break
// 		}

// 		if send {
// 			pub.SendMessage(reqID, strings.Join(append(payloadSend, payloadRecv...), "|")) //REFACTOR
// 		}
// 	}

// 	forceClose <- true
// 	fmt.Println(userID, "closed")

// }
