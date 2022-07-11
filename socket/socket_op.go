package socket

import (
	"fmt"
	"sync/atomic"
)

func socket_op(router_conn *router_conn_obj, op byte, b []byte, received *uint32) {

	if op == 50 || op == 51 {
		if op == 51 {
			atomic.AddUint32(received, 1)
		}

		param, payload, err := byte_to_params_and_payload(b, 1, true)
		if err != nil {
			log_err("socket_op:" + err.Error())
			return
		}

		fmt.Println("WS WSID, OP, AND PAYLOAD:", param[0], payload[0], string(payload[1:]))

		exists := false

		shard := ws_id_chan_table.get_shard(param[0])

		shard.RLock()

		if shard.table[param[0]] != nil {
			shard.table[param[0]] <- payload
			exists = true
		}

		shard.RUnlock()

		if !exists {
			switch payload[0] {
			case 31:
				param, _, err = byte_to_params_and_payload(payload[1:], 2, false)
				if err != nil {
					log_err("socket_op:31" + err.Error())
					return
				}
				get_prev_state(param[0], param[1], false, []byte("{}"))
			}
		}

	}

}
