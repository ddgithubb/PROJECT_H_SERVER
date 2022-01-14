package schemas

// UserByUsernameSchema struct
type UserByUsernameSchema struct {
	Username string
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
	Username   string
	RelationID string
	ChainID    string
	LastSeen   int64
	LastRecv   int64
	Key        int
}

// RequestsSchema struct
type RequestsSchema struct {
	Username   string
	RelationID string
	Requested  bool
}
