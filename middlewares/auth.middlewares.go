package middlewares

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/schemas"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// Authenticate authenticates refresh and access tokens
func Authenticate(c *fiber.Ctx) error {

	var sessionInfo schemas.TokensInfoSchema
	sessionInfo.SessionID = string(c.Request().Header.Peek("x-session-id"))
	sessionInfo.RefreshToken.Token = string(c.Request().Header.Peek("x-refresh-token"))
	refresh := string(c.Request().Header.Peek("x-refresh"))
	expireAt, err := helpers.ParseStringToInt(string(c.Request().Header.Peek("x-refresh-token-expire")))
	if err != nil || sessionInfo.SessionID == "" || sessionInfo.RefreshToken.Token == "" {
		return errors.HandleUnauthorizedError(c)
	}
	sessionInfo.RefreshToken.ExpireAt = expireAt

	fmt.Println("AUTHENTICATING " + sessionInfo.SessionID + "; PATH " + c.Path())

	authorization := c.Request().Header.Peek("Authorization")
	accessToken := strings.Split(string(authorization), "Bearer ")[1]

	if time.Unix(sessionInfo.RefreshToken.ExpireAt, 0).Before(time.Now().UTC()) {
		return errors.HandleBadRequestError(c, "RefreshToken", "expired")
	}

	userID, username, err := helpers.ParseJWT(c, accessToken)
	if userID == "expired" {
		res, err := global.RedisClient.HGetAll(global.Context, "refreshtokens:"+sessionInfo.SessionID).Result()
		if err != nil {
			return errors.HandleInternalError(c, "get_refresh_tokens", "Redis: "+err.Error())
		}

		if _, ok := res["token"]; !ok {
			return errors.HandleInvalidRequestError(c, "RefreshToken", "invalid")
		}

		userID = res["userid"]

		if refresh == "true" {
			if err = helpers.GenerateAndRefreshTokens(c, userID, sessionInfo.SessionID, username, sessionInfo.RefreshToken.Token != res["token"]); err != nil {
				return err
			}
		}
	}

	if userID == "" {
		return err
	}

	c.Locals("userid", userID)
	return c.Next()
}

// AuthenticateStream authenticates websocket connection
func AuthenticateStream(c *fiber.Ctx) error {

	if websocket.IsWebSocketUpgrade(c) {
		accessToken := c.Query("token")

		userID, username, err := helpers.ParseJWT(c, accessToken)
		if userID == "expired" {
			return errors.HandleInvalidRequestError(c, "AccessToken", "expired")
		} else if userID == "" {
			return err
		}

		c.Locals("userid", userID)
		c.Locals("username", username)
		return c.Next()
	}

	return errors.HandleInternalError(c, "websocket_upgrade", fiber.ErrUpgradeRequired.Error())
}
