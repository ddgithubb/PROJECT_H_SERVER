package schemas

// MessageSchema struct
type MessageSchema struct {
	MessageID string
	UserID    string
	Created   int64
	Duration  int
	Seen      bool
	Action    int
	Display   string
}
