package messages

type op_type int

type ws_message struct {
	Op        op_type
	OriginID  string
	TargetID  string
	Timestamp int64
	Signature string
	Atomic    bool
	Data      interface{}
}

func construct_ws_message(op op_type, originID string, targetID string, timestamp int64, signature string, atomic bool, data interface{}) ws_message {
	return ws_message{
		Op:        op,
		OriginID:  originID,
		TargetID:  targetID,
		Timestamp: timestamp,
		Signature: signature,
		Atomic:    atomic,
		Data:      data,
	}
}

//////////////////////////////////////// WEBSCOKET MESSAGE OP DATA ////////////////////////////////////////

type friend_request_data struct {
	OriginUsername string
	TargetUsername string
}

type friend_accept_data struct {
	ChainID string
	Created int64
}

type message_data struct {
	ChainID   string
	MessageID string
	Created   int64
	Expires   int64
	Type      int
	Seen      bool
	Display   string
	Duration  int64
}
