package socket

import (
	"sync"
	"time"

	"github.com/aidarkhanov/nanoid/v2"
	"github.com/segmentio/fasthash/fnv1a"
)

const CONCURRENCY = 32
const VALID_NANOID_CHAR = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const MAX_CLIENT_RESPONSE = 10 * time.Second
const MAX_PREV_STATE_RESPONSE = 1 * time.Second
const MAX_WS_CONNECTION_TIME = 1 * time.Hour

type conc_ws_id_table struct {
	table map[string]chan []byte
	sync.RWMutex
}
type conc_ws_id_table_shards []*conc_ws_id_table

func (ct conc_ws_id_table_shards) get_shard(id string) *conc_ws_id_table {
	return ct[fnv1a.HashString32(id)%CONCURRENCY]
}

var ws_id_chan_table conc_ws_id_table_shards = func() conc_ws_id_table_shards {
	shards := make([]*conc_ws_id_table, CONCURRENCY)

	for i := 0; uint32(i) < CONCURRENCY; i++ {
		shards[i] = &conc_ws_id_table{table: make(map[string]chan []byte)}
	}

	return shards
}()

func create_connection() (string, chan []byte) {

	messageChannel := make(chan []byte, 4)

	WSID, err := nanoid.GenerateString(VALID_NANOID_CHAR, 10)
	if err != nil {
		return "", nil
	}

	shard := ws_id_chan_table.get_shard(WSID)
	exists := true

	shard.Lock()

	for {
		_, exists = shard.table[WSID]
		if exists {
			WSID, err = nanoid.GenerateString(VALID_NANOID_CHAR, 10)
			if err != nil {
				return "", nil
			}
		} else {
			break
		}
	}

	shard.table[WSID] = messageChannel

	shard.Unlock()

	return WSID, messageChannel
}

func delete_connection(WSID string) {

	shard := ws_id_chan_table.get_shard(WSID)

	shard.Lock()

	msg_chan := shard.table[WSID]
	delete(shard.table, WSID)

	shard.Unlock()

	if msg_chan != nil {
		close(msg_chan)
	}

}

func add_user_session(userID string, sessionID string) (string, chan []byte) {

	WSID, messageChannel := create_connection()

	socket_write(20, nil, userID, sessionID, WSID)

	return WSID, messageChannel
}

func recover_user_session(userID string, sessionID string, prevWSID string) (string, chan []byte) {

	WSID, messageChannel := create_connection()

	socket_write(21, nil, userID, sessionID, WSID, prevWSID)

	return WSID, messageChannel
}

func remove_user_session(userID string, sessionID string, WSID string) {

	delete_connection(WSID)

	socket_write(22, nil, userID, sessionID, WSID)
}

func get_prev_state(new_sock_id string, new_ws_id string, success bool, state []byte) {

	success_text := "1"
	if !success {
		success_text = "0"
	}

	socket_write(30, state, new_sock_id, new_ws_id, success_text)

}
