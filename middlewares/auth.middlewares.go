package middlewares

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/schemas"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
)

// Authenticate authenticates refresh and access tokens
func Authenticate(c *fiber.Ctx) error {

	var sessionInfo schemas.TokensInfoSchema
	sessionInfo.SessionID = string(c.Request().Header.Peek("x-session-id"))
	sessionInfo.RefreshToken.Token = string(c.Request().Header.Peek("x-refresh-token"))
	refresh := string(c.Request().Header.Peek("x-refresh"))
	expireAt, err := helpers.ParseStringToInt(string(c.Request().Header.Peek("x-refresh-token-expire")))
	if err != nil || sessionInfo.SessionID == "" || sessionInfo.RefreshToken.Token == "" {
		return errors.HandleBadRequestError(c, "Auth", "empty")
	}
	sessionInfo.RefreshToken.ExpireAt = expireAt

	fmt.Println("AUTHENTICATING " + sessionInfo.SessionID + "; PATH " + c.Path())

	authorization := c.Request().Header.Peek("Authorization")
	bearerSplit := strings.Split(string(authorization), "Bearer ")
	accessToken := ""

	if len(bearerSplit) == 2 {
		accessToken = bearerSplit[1]
	} else {
		return errors.HandleBadRequestError(c, "AccessToken", "invalid")
	}

	if time.UnixMilli(sessionInfo.RefreshToken.ExpireAt).Before(time.Now().UTC()) {
		return errors.HandleBadRequestError(c, "RefreshToken", "expired")
	}

	expired, userID, username, sessionID, err := helpers.ParseJWT(c, accessToken)
	if err != nil {
		return err
	}

	fmt.Println(userID, sessionID, sessionInfo.SessionID)

	if sessionID != sessionInfo.SessionID {
		return errors.HandleBadRequestError(c, "SessionID", "invalid")
	}

	if expired {
		res, err := global.RedisClient.HGetAll(global.Context, "refreshtokens:"+sessionInfo.SessionID).Result()
		if err != nil {
			if err == redis.Nil {
				return errors.HandleInvalidRequestError(c, "RefreshToken", "invalid")
			}
			return errors.HandleInternalError(c, "get_refresh_tokens", "Redis: "+err.Error())
		}

		if userID != res["userid"] {
			return errors.HandleBadRequestError(c, "UserID", "Invalid")
		}

		if refresh == "true" {
			if err = helpers.GenerateAndRefreshTokens(c, userID, sessionInfo.SessionID, username); err != nil {
				return err
			}
		}
	}

	c.Locals("userid", userID)
	c.Locals("username", username)
	c.Locals("sessionid", sessionID)
	return c.Next()
}
