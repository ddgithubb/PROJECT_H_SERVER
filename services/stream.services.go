package services

import (
	"PROJECT_H_server/config"
	"PROJECT_H_server/errors"
	"PROJECT_H_server/helpers"
	Errors "errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/pebbe/zmq4"
)

// Stream starts and maintains websocket connection
func Stream(ws *websocket.Conn) {

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

	var (
		mt       int
		msg      []byte
		req      string
		reqChunk []string
		payload  []string
		send     bool
	)
	for {
		if err = ws.SetReadDeadline(time.Now().Add(time.Second * 190)); err != nil {
			errors.HandleWebsocketError(ws, "websocket_read_deadline", err.Error())
			break
		}
		if mt, msg, err = ws.ReadMessage(); err != nil {
			if err != websocket.ErrCloseSent && !websocket.IsCloseError(err, 1000) && !strings.Contains(err.Error(), "i/o timeout") && !strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host") {
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

		if req == "PING" {
			if err = ws.WriteMessage(websocket.TextMessage, []byte("PONG"+fmt.Sprint(version))); err != nil {
				errors.HandleWebsocketError(ws, "websocket_write_PONG", err.Error())
				break
			}
			continue
		}

		reqChunk = strings.Split(req, "|")
		payload = []string{reqChunk[0]}
		send = true

		switch reqChunk[0] {
		case "request":
			payload = append(payload, userID, username, reqChunk[2])
		case "unrequest":
			payload = append(payload, userID)
		case "accept":
			payload = append(payload, userID)
		case "unfriend":
			payload = append(payload, userID)
		case "send-message":
			payload = append(payload, userID, reqChunk[2], reqChunk[3], reqChunk[4], reqChunk[5])
		case "send-action":

			err = helpers.UpdateAction(reqChunk[2], reqChunk[3], reqChunk[4])

			if err != nil {
				break
			}

			payload = append(payload, userID, reqChunk[3], reqChunk[4])
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
