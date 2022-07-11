package helpers

import (
	"PROJECT_H_server/config"
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/schemas"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
)

// GenerateJWT generates a jwt token with a claim
func GenerateJWT(c *fiber.Ctx, userID string, username string, sessionID string) (string, error) {
	exp := time.Now().Add(time.Hour * 1).Unix()
	user := jwt.MapClaims{}
	user["id"] = userID
	user["username"] = username
	user["session_id"] = sessionID
	user["exp"] = exp
	jt := jwt.NewWithClaims(jwt.SigningMethodRS256, user)
	token, err := jt.SignedString(global.JwtKey)
	if err != nil {
		return "", errors.HandleInternalError(c, "jwt", "jwt: "+err.Error())
	}
	return token, nil
}

// ParseJWT parses a jwt to userID
func ParseJWT(c *fiber.Ctx, jwtString string) (bool, string, string, string, error) {
	expired := false
	token, err := jwt.Parse(jwtString, func(token *jwt.Token) (interface{}, error) {
		return global.JwtParseKey, nil
	})
	if err != nil {
		if err.(*jwt.ValidationError).Errors == jwt.ValidationErrorExpired {
			expired = true
		} else {
			return expired, "", "", "", errors.HandleInternalError(c, "jwt_parse", err.Error())
		}
	}
	user, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return expired, "", "", "", errors.HandleInternalError(c, "jwt_claims", "jwt_claims not valid")
	}
	return expired, user["id"].(string), user["username"].(string), user["session_id"].(string), nil
}

// GenerateAndRefreshTokens generates and interacts with redis to store tokens and then sets response header
func GenerateAndRefreshTokens(c *fiber.Ctx, userID string, sessionID string, username string) error {

	var tokens schemas.TokensSchema
	redisError := false

	_, err := global.RedisClient.Pipelined(global.Context, func(pipe redis.Pipeliner) error {

		var err error

		tokens.RefreshToken.Token, err = RandomTokenString(40)
		if err != nil {
			redisError = true
			return errors.HandleInternalError(c, "password", "hex token error")
		}

		tokenExpire := time.Now().Add(global.RefreshTokenDuration)
		tokens.RefreshToken.ExpireAt = tokenExpire.UnixMilli()

		query := map[string]interface{}{
			"token":  tokens.RefreshToken.Token,
			"userid": userID,
			"ip":     c.IP(),
		}

		err = pipe.HSet(global.Context, "refreshtokens:"+sessionID, query).Err()
		if err != nil {
			redisError = true
			return errors.HandleInternalError(c, "set_refresh_tokens", "Redis: "+err.Error())
		}
		err = pipe.ExpireAt(global.Context, "refreshtokens:"+sessionID, tokenExpire).Err()
		if err != nil {
			redisError = true
			return errors.HandleInternalError(c, "expire_refresh_tokens", "Redis: "+err.Error())
		}

		tokens.AccessToken, err = GenerateJWT(c, userID, username, sessionID)
		if tokens.AccessToken == "" {
			redisError = true
			return err
		}

		return nil
	})

	if err != nil {
		return errors.HandleInternalError(c, "pipeline", "Redis: "+err.Error())
	}
	if redisError {
		return err
	}

	c.Response().Header.Add("x-refreshed", "true")
	c.Response().Header.Add("x-refresh-token", tokens.RefreshToken.Token)
	c.Response().Header.Add("x-refresh-token-expire", strconv.FormatInt(tokens.RefreshToken.ExpireAt, 10))
	c.Response().Header.Add("x-access-token", tokens.AccessToken)
	return nil
}

// SendVerifEmail send a verification email
func SendVerifEmail(c *fiber.Ctx, email string, code string) {
	// emailMsg := mail.NewMSG()
	// emailMsg.SetFrom(config.Config.EmailFrom + " <" + config.Config.SMTP.User + ">").AddTo(email).SetSubject("Verification code")
	// emailMsg.SetBody(mail.TextHTML, "<html><body><div><h1>Your verification code is: <b>"+code+"</b></h1><br><p>Please enter the code as instructed in the app within <b>24 hours</b></p></div></body></html>")
	// err := emailMsg.Send(global.EmailClient)
	// if err != nil {
	// 	global.Logger.Println("Email sender error: " + err.Error())
	// }

	from := "From: " + config.Config.EmailFrom + "\n"
	subject := "Subject: Verification code\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := "<html><body><div><h1>Your code is: <b>" + code + "</b></h1><br><p>Please enter the code as instructed in the app within <b>24 hours</b></p></div></body></html>"

	err := EmailSender(email, from+subject+mime+body)
	if err != nil {
		global.InternalLogger.Println("Email sender error: " + err.Error())
	}
}

// DecryptRequestPassword decrypts password from request
func DecryptRequestPasswordHash(c *fiber.Ctx, encPasswordHash string) ([]byte, error) {
	chunks := strings.Split(encPasswordHash, ".")
	if len(chunks) != 3 {
		return []byte{}, errors.HandleInternalError(c, "invalid_chunks", strconv.Itoa(len(chunks)))
	}
	nonce, err := base64.StdEncoding.DecodeString(chunks[2])
	if err != nil {
		return []byte{}, errors.HandleInternalError(c, "base64_decoding", err.Error())
	}
	numTime, err := strconv.ParseInt(string(nonce), 10, 64)
	if err != nil {
		return []byte{}, errors.HandleInternalError(c, "string_time_to_string", err.Error())
	}
	timestamp := MilisecondsToTime(numTime)
	if err != nil {
		return []byte{}, errors.HandleInternalError(c, "number_time_to_time", err.Error())
	}
	if timestamp.Add(60 * time.Second).Before(time.Now().UTC()) {
		return []byte{}, errors.HandleBadRequestError(c, "EncPasswordHash", "Invalid")
	}
	encKey, err := base64.StdEncoding.DecodeString(chunks[0])
	if err != nil {
		return []byte{}, errors.HandleInternalError(c, "base64_decoding", err.Error())
	}
	encPassword, err := base64.StdEncoding.DecodeString(chunks[1])
	if err != nil {
		return []byte{}, errors.HandleInternalError(c, "base64_decoding", err.Error())
	}
	key, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, global.PrivateKey, encKey, nil)
	if err != nil {
		return []byte{}, errors.HandleInternalError(c, "decrypt_oaep", err.Error())
	}
	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, errors.HandleInternalError(c, "new_cipher", err.Error())
	}
	cipherAESGCM, err := cipher.NewGCM(cipherBlock)
	if err != nil {
		return []byte{}, errors.HandleInternalError(c, "cipher_GCM", err.Error())
	}
	minNonce := []byte(chunks[2])[len(chunks[2])-12:]
	passwordHash, err := cipherAESGCM.Open(nil, minNonce, encPassword, nil)
	if err != nil {
		return []byte{}, errors.HandleInternalError(c, "decrypt_cipher_GCM", err.Error())
	}
	return passwordHash, nil
}
