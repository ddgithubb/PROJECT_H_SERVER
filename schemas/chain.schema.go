package schemas

// ChainSchema struct
type ChainSchema struct {
	MessageID string
	UserID    string
	Created   int64
	Duration  int
	Seen      bool
	Action    int
	Display   string
}
