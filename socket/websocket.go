package socket

import (
	"PROJECT_H_server/global"
	er "errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt"
	jsoniter "github.com/json-iterator/go"
)

func write_message(ws *websocket.Conn, code op_type, data interface{}) error {

	b, err := jsoniter.Marshal(construct_ws_message(code, data))
	if err != nil {
		handleWebsocketError(ws, "jsoniter_marshal", err.Error())
	}

	fmt.Println("SEND WS:", string(b))

	return ws.WriteMessage(websocket.TextMessage, b)
}

func send_comm_msg(msg_chan chan []byte, msg string) {

	b := make([]byte, 1+len(msg))
	b[0] = 0
	copy(b[1:], msg)

	msg_chan <- b

}

func handleWebsocketError(c *websocket.Conn, problem string, err string) {
	websocket_logger.Println("ip: " + c.RemoteAddr().String() + "; Problem: " + problem + "; Error: " + err)
}

func ClientSocket(ws *websocket.Conn) {

	defer func() {
		if ws != nil && ws.Conn != nil {
			ws.Close()
		}
	}()

	userID, sessionID, prevWSID, err := auth_client(ws)
	if err != nil {
		return
	}

	msg_chan, WSID, prev_state, refresh, err_func := connect_user(ws, userID, sessionID, prevWSID)

	var gate_state int32

	atomic.StoreInt32(&gate_state, 0)

	go msg_chan_recv(ws, msg_chan, userID, sessionID, WSID, &gate_state, prev_state.LastTimestamp, prev_state.MQ)

	err = write_message(ws, 1002, auth_response_data{
		WSID:    WSID,
		Refresh: refresh,
	})
	if err != nil {
		err_func(ws, "write op:1002", err.Error())
		return
	}

	var (
		mt int
		b  []byte
	)
	for {

		if err = ws.SetReadDeadline(time.Now().Add(MAX_WS_CONNECTION_TIME)); err != nil {
			err_func(ws, "websocket_read_deadline", err.Error())
			break
		}

		if mt, b, err = ws.ReadMessage(); err != nil {
			fmt.Println("WEBSOCKET READ ERR with error : ", err, websocket.IsCloseError(err, 1000), websocket.IsCloseError(err, 1006))
			if websocket.IsCloseError(err, 1006) || strings.Contains(err.Error(), "i/o timeout") {
				remove_user_session(userID, sessionID, WSID)
			} else {
				send_comm_msg(msg_chan, "SET_TIMER")
			}
			break
			// TEST what happesn aftere a while (before max-ws-conn-time), to see if it automatically forms err
			// Test network disconnect
			// 1006 is user completely closes client (memory completely deteriorated, should break)
			// 1000 is normal closure by JS client (should not break!)
			// "An existing connection was forcibly closed by the remote host."
			//err != websocket.ErrCloseSent && !websocket.IsCloseError(err, 1000)
		}
		if mt == websocket.BinaryMessage {
			err_func(ws, "websocket_read", "binary message")
			break
		}

		fmt.Println("RECV WS:", string(b))

		switch jsoniter.Get(b, "Op").ToInt() {
		case 1000:
			write_message(ws, 1000, nil)
		case 1005:
			atomic.StoreInt32(&gate_state, 0)
		case 1006:
			send_comm_msg(msg_chan, "FLUSH")
			atomic.StoreInt32(&gate_state, 1)
		case 1007:
			data := new(ack_data)
			jsoniter.Get(b, "Data").ToVal(data)
			send_comm_msg(msg_chan, data.Signature+strconv.FormatInt(data.Timestamp, 10))
		default:
			handleWebsocketError(ws, "op_code", "unrecognized")
		}
	}
}

func msg_chan_recv_err(ws *websocket.Conn, op byte, err string) {
	handleWebsocketError(ws, "msg_chan_recv op:"+fmt.Sprint(op), err)
	if ws != nil && ws.Conn != nil {
		ws.Close()
	}
}

func msg_chan_recv(ws *websocket.Conn, msg_chan chan []byte, userID, sessionID, WSID string, gate_state *int32, last_timestamp uint64, message_queue map[string][]byte) {

	var b []byte
	var ok bool
	var param []string
	var payload []byte
	var err error
	var msg string

	var temp_gate int32
	var temp_op int
	var temp_timestamp uint64
	var temp_mq_slice [][]byte
	var signature string

	for {
		b, ok = <-msg_chan

		if !ok {
			//fmt.Println("CLSOING MSG_CHAN_RECV WITH WSID:", WSID)
			return
		}

		if b[0] == 0 {
			switch string(b[1:]) {
			case "FLUSH":
				if len(message_queue) == 0 {
					break
				}

				temp_mq_slice = make([][]byte, len(message_queue))
				i := 0
				for _, v := range message_queue {
					temp_mq_slice[i] = v
					i++
				}

				sort.Slice(temp_mq_slice, func(i, j int) bool {
					return jsoniter.Get(temp_mq_slice[i], "Timestamp").ToUint64() < jsoniter.Get(temp_mq_slice[j], "Timestamp").ToUint64()
				})

				for i := 0; i < len(temp_mq_slice); i++ {
					err = ws.WriteMessage(websocket.TextMessage, temp_mq_slice[i])
					if err != nil {
						handleWebsocketError(ws, "FLUSH", err.Error())
						return
					}
				}
			case "SET_TIMER":
				go func() {
					time.Sleep(MAX_WS_CONNECTION_TIME)
					remove_user_session(userID, sessionID, WSID)
				}()
			default:
				delete(message_queue, msg)
			}
		}

		switch b[0] {
		case 30:
			param, _, err = byte_to_params_and_payload(b[1:], 2, false)
			if err != nil {
				msg_chan_recv_err(ws, 30, err.Error())
			}

			state, err := jsoniter.Marshal(ws_state{
				LastTimestamp: last_timestamp,
				MQ:            message_queue,
			})
			if err != nil {
				msg_chan_recv_err(ws, 30, err.Error())
			}

			get_prev_state(param[0], param[1], true, state)

			if ws != nil && ws.Conn != nil {
				ws.Close()
			}
		case 32:
			if ws != nil && ws.Conn != nil {
				ws.Close()
			}
		case 100:
			_, payload, err = byte_to_params_and_payload(b[1:], 0, true)
			if err != nil {
				msg_chan_recv_err(ws, 100, err.Error())
			}

			temp_timestamp = jsoniter.Get(payload, "Timestamp").ToUint64()
			signature = jsoniter.Get(payload, "Signature").ToString()
			temp_gate = atomic.LoadInt32(gate_state)

			if temp_timestamp <= last_timestamp && !jsoniter.Get(payload, "Atomic").ToBool() && temp_gate == 1 {

				temp_op = jsoniter.Get(payload, "Op").ToInt()

				if temp_op >= 200 && temp_op < 300 {
					write_message(ws, 3101, nil)
				} else if temp_op >= 300 && temp_op < 400 {
					write_message(ws, 3102, refresh_chain_data{
						ChainID: jsoniter.Get(payload, "Data", "ChainID").ToString(),
					})
				}

			} else {

				if temp_timestamp > last_timestamp {
					last_timestamp = temp_timestamp
				}

				message_queue[signature+strconv.FormatUint(temp_timestamp, 10)] = payload

				if temp_gate == 1 {
					err = ws.WriteMessage(websocket.TextMessage, payload)
					if err != nil {
						msg_chan_recv_err(ws, 100, err.Error())
						return
					}
				}

			}
		}
	}

}

func auth_client(ws *websocket.Conn) (string, string, string, error) {

	var msg []byte
	var err error
	var tries byte = 0

	for {
		tries++
		err = write_message(ws, 1001, nil)
		if err != nil {
			handleWebsocketError(ws, "websocket_write", err.Error())
			return "", "", "", err
		}

		if err = ws.SetReadDeadline(time.Now().Add(MAX_CLIENT_RESPONSE)); err != nil {
			write_message(ws, 3000, nil)
			return "", "", "", err
		}

		if _, msg, err = ws.ReadMessage(); err != nil {
			handleWebsocketError(ws, "websocket_read", err.Error())
			return "", "", "", err
		}

		if jsoniter.Get(msg, "Op").ToInt() != 1001 {
			handleWebsocketError(ws, "op:1001", "expected 1001, receieved:"+fmt.Sprint(jsoniter.Get(msg, "Op").ToInt()))
			return "", "", "", er.New("expected 1001, receieved:" + fmt.Sprint(jsoniter.Get(msg, "Op").ToInt()))
		}

		res := new(auth_token_data)
		jsoniter.Get(msg, "Data").ToVal(res)

		if res.Token == "" {
			handleWebsocketError(ws, "auth_token", "empty token")
			return "", "", "", er.New("empty token")
		}

		token, err := jwt.Parse(string(res.Token), func(token *jwt.Token) (interface{}, error) {
			return global.JwtParseKey, nil
		})
		if err != nil {
			if err.(*jwt.ValidationError).Errors == jwt.ValidationErrorExpired {
				write_message(ws, 3001, nil)
				if tries == 2 {
					handleWebsocketError(ws, "auth_token", err.Error())
					return "", "", "", err
				} else {
					continue
				}
			} else {
				handleWebsocketError(ws, "jwt_parse_error", err.Error())
				return "", "", "", err
			}
		}

		user, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			handleWebsocketError(ws, "jwt_claims", "invalid claims")
			return "", "", "", er.New("invalid claims")
		}

		return user["id"].(string), user["session_id"].(string), res.PrevWSID, nil
	}

}

func connect_user(ws *websocket.Conn, userID, sessionID, prevWSID string) (msg_chan chan []byte, WSID string, prev_state *ws_state, refresh bool, err_func func(ws *websocket.Conn, problem string, err string)) {

	prev_state = new(ws_state)
	refresh = true

	if prevWSID == "" {
		WSID, msg_chan = add_user_session(userID, sessionID)
	} else {
		WSID, msg_chan = recover_user_session(userID, sessionID, prevWSID)
	}

	err_func = func(ws *websocket.Conn, problem string, err string) {
		handleWebsocketError(ws, problem, err)
		remove_user_session(userID, sessionID, WSID)
	}

	if prevWSID == "" {

		b := <-msg_chan

		if b[0] != 20 {
			err_func(ws, "add_user_session", "response not op:20, but is op:"+fmt.Sprint(b[0]))
			return
		}
		param, _, err := byte_to_params_and_payload(b[1:], 1, false)
		if err != nil {
			err_func(ws, "add_user_session", err.Error())
			return
		} else if param[0] != "1" {
			err_func(ws, "add_user_session", "unsuccessful")
			return
		}

	} else {

		b := <-msg_chan

		if b[0] != 21 {
			err_func(ws, "recover_user_session", "response not op:21, but is op:"+fmt.Sprint(b[0]))
			return
		}
		param, _, err := byte_to_params_and_payload(b[1:], 1, false)
		if err != nil {
			err_func(ws, "recover_user_session", err.Error())
			return
		}
		if param[0] == "1" {
			b = <-msg_chan

			if b[0] != 31 {
				err_func(ws, "recv_prev_state", "response not op:31, but is op:"+fmt.Sprint(b[0]))
				return
			}
			param, payload, err := byte_to_params_and_payload(b[1:], 1, true)
			if err != nil {
				err_func(ws, "recv_prev_state", err.Error())
				return
			}
			if param[0] == "1" {
				refresh = false

				err = jsoniter.Unmarshal(payload, prev_state)
				if err != nil {
					handleWebsocketError(ws, "jsoniter_unmarshal", err.Error())
				}
			}
		}
	}

	if prev_state.MQ == nil {
		prev_state.MQ = make(map[string][]byte)
	}

	return
}
