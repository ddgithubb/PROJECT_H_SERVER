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
	RelationID string
	Username   string
	ChainID    string
	Created    int64
	LastSeen   int64
	LastRecv   int64
	Key        int
}

// RequestsSchema struct
type RequestsSchema struct {
	Username   string
	RelationID string
	Created    int64
	Requested  bool
}
