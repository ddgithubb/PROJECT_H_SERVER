package schemas

// MessageSchema struct
type MessageSchema struct {
	MessageID string
	UserID    string
	Created   int64
	Expires   int64
	Type      int
	Seen      bool
	Display   string
	Duration  int
}
