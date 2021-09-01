package schemas

// UserByUsernameSchema struct
type UserByUsernameSchema struct {
	Username string
}

// LoginResponseSchema struct
type LoginResponseSchema struct {
	Data      UserInfoSchema
	SessionID string
	Tokens    TokensSchema
}

// UserInfoSchema struct
type UserInfoSchema struct {
	UserID    string
	Username  string
	Statement string
	Relations RelationsSchema
}

// PublicUserSchema struct
type PublicUserSchema struct {
	Username  string
	UserID    string
	Statement string
}

// RelationsSchema struct
type RelationsSchema struct {
	Friends   []FriendsSchema
	Requests  []RequestsSchema
	Requested []RequestsSchema
}

// FriendsSchema struct
type FriendsSchema struct {
	Username string
	UserID   string
	ChainID  string
	LastSeen int64
}

// RequestsSchema struct
type RequestsSchema struct {
	Username  string
	UserID    string
	Requested bool
}

// SendAudioSchema struct
type SendAudioSchema struct {
	ChainID string `form:"cequestID"`
	Audio   []byte `form:"audio"`
}
