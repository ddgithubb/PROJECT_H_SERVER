package socket

type op_type int

type ws_message struct {
	Op   op_type
	Data interface{}
}

func construct_ws_message(op op_type, data interface{}) ws_message {
	return ws_message{
		Op:   op,
		Data: data,
	}
}

type ws_state struct {
	LastTimestamp uint64
	MQ            map[string][]byte
}

//////////////////////////////////////// WEBSCOKET SERVER OP DATA ////////////////////////////////////////

// 1002
type auth_response_data struct {
	WSID    string
	Refresh bool
}

//////////////////////////////////////// WEBSCOKET CLIENT OP DATA ////////////////////////////////////////

// 1001
type auth_token_data struct {
	Token    string
	PrevWSID string
}

// 1007
type ack_data struct {
	Timestamp int64
	Signature string
}

//////////////////////////////////////// WEBSCOKET ERROR OP DATA ////////////////////////////////////////

// 3102
type refresh_chain_data struct {
	ChainID string
}
