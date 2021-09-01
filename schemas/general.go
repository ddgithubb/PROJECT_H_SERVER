package schemas

// ErrorResponse struct
type ErrorResponse struct {
	Error   bool
	Type    string
	Problem string
}

// Message struct
type Message struct {
	Message string
}

// Response struct
type DataResponse struct {
	Refreshed bool
	Tokens    TokensSchema
	Data      interface{}
}
