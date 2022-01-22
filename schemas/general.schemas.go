package schemas

// ErrorResponse struct
type ErrorResponse struct {
	Error       bool
	Problem     string
	Description string
}

// Message struct
type Message struct {
	Message string
}
