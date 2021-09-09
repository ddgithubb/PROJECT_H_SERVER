package schemas

// EmailSchema struct
type EmailSchema struct {
	Email string `validate:"required,email,max=1000"`
}

// RegisterSchema struct
type RegisterSchema struct {
	Username        string `validate:"required,max=30"`
	Email           string `validate:"required,email,max=1000"`
	EncPasswordHash string `validate:"required,min=8"`
}

// VerifyEmailSchema struct
type VerifyEmailSchema struct {
	Email string `validate:"required,email,max=1000"`
	Code  string `validate:"required,len=6"`
}

// LoginSchema struct
type LoginSchema struct {
	Email           string `validate:"required,email,max=1000"`
	EncPasswordHash string `validate:"required"`
	DeviceToken     string `validate:"required"`
}

// RefreshTokenSchema struct
type RefreshTokenSchema struct {
	Token    string `validate:"required"`
	ExpireAt int64  `validate:"required"`
}

// TokenSchema struct
type TokensSchema struct {
	RefreshToken RefreshTokenSchema
	AccessToken  string
}

// TokensInfoSchema struct
type TokensInfoSchema struct {
	SessionID    string             `validate:"required" json:"sessionID" form:"sessionID"`
	RefreshToken RefreshTokenSchema `validate:"dive" json:"refreshToken"`
}
