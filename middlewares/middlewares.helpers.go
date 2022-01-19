package middlewares

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/schemas"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// Authenticate handles authentication with JSON request
func Authenticate(c *fiber.Ctx) error {

	req := new(schemas.TokensInfoSchema)

	if err := c.BodyParser(req); err != nil {
		return errors.HandleBadJsonError(c)
	}

	if err := global.Validator.Struct(req); err != nil {
		return errors.HandleValidatorError(c, err)
	}

	return AuthenticateHelper(c, req)
}

// AuthenticateForm handles authentication with form data request
func AuthenticateForm(c *fiber.Ctx) error {

	req := new(schemas.TokensInfoSchema)

	form, err := c.MultipartForm()
	if err != nil {
		return errors.HandleBadRequestError(c, "Multipart", "invalid")
	}

	temp := form.Value["sessionID"]
	if len(temp) == 0 {
		return errors.HandleBadRequestError(c, "SessionID", "missing")
	}

	req.SessionID = temp[0]

	temp = form.Value["token"]
	if len(temp) == 0 {
		return errors.HandleBadRequestError(c, "Token", "missing")
	}

	req.RefreshToken.Token = temp[0]

	temp = form.Value["expireAt"]
	if len(temp) == 0 {
		return errors.HandleBadRequestError(c, "expireAt", "missing")
	}

	req.RefreshToken.ExpireAt, err = strconv.ParseInt(temp[0], 10, 64)
	if err != nil {
		return errors.HandleBadRequestError(c, "expireAt", "invalid")
	}

	c.Locals("multipart", form)

	if err := global.Validator.Struct(req); err != nil {
		return errors.HandleValidatorError(c, err)
	}

	return AuthenticateHelper(c, req)
}

// AuthenticateHelper includes the logic for refreshing the tokens
func AuthenticateHelper(c *fiber.Ctx, req *schemas.TokensInfoSchema) error {

	authorization := c.Request().Header.Peek("Authorization")
	accessToken := strings.Split(string(authorization), "Bearer ")[1]

	if time.Unix(req.RefreshToken.ExpireAt, 0).Before(time.Now().UTC()) {
		return errors.HandleBadRequestError(c, "Refresh token", "expired")
	}

	c.Locals("data", "")
	response := schemas.DataResponse{
		Refreshed: false,
	}

	userID, err := helpers.ParseJWT(c, accessToken)
	if userID == "expired" {
		res, err := global.RedisClient.HGetAll(global.Context, "refreshtokens:"+req.SessionID).Result()
		if err != nil {
			return errors.HandleInternalError(c, "get_refresh_tokens", "Redis: "+err.Error())
		}

		if _, ok := res["token"]; !ok {
			return errors.HandleInvalidRequestError(c, "Refresh Token", "invalid")
		}

		var tokens schemas.TokensSchema
		redisError := false

		_, err = global.RedisClient.Pipelined(global.Context, func(pipe redis.Pipeliner) error {

			if req.RefreshToken.Token != res["token"] {
				err = pipe.Del(global.Context, "refreshtokens:"+req.SessionID).Err()
				if err != nil {
					redisError = true
					return errors.HandleInternalError(c, "refresh_tokens", "Redis: "+err.Error())
				}
				redisError = true
				return errors.HandleInvalidRequestError(c, "Refresh Token", "invalid")
			}

			userID = res["userid"]

			tokens.RefreshToken.Token, err = helpers.RandomTokenString(40)
			if err != nil {
				redisError = true
				return errors.HandleInternalError(c, "password", "hex token error")
			}

			tokens.RefreshToken.ExpireAt = time.Now().UTC().Add(global.RefreshTokenDuration).Unix()

			query := map[string]interface{}{
				"token":  tokens.RefreshToken.Token,
				"userid": userID,
				"ip":     c.IP(),
			}

			err = pipe.HSet(global.Context, "refreshtokens:"+req.SessionID, query).Err()
			if err != nil {
				redisError = true
				return errors.HandleInternalError(c, "set_refresh_tokens", "Redis: "+err.Error())
			}
			err = pipe.Expire(global.Context, "refreshtokens:"+req.SessionID, global.RefreshTokenDuration).Err()
			if err != nil {
				redisError = true
				return errors.HandleInternalError(c, "expire_refresh_tokens", "Redis: "+err.Error())
			}

			tokens.AccessToken, err = helpers.GenerateJWT(c, userID)
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
		response.Tokens = tokens
		response.Refreshed = true
	}

	if userID == "" {
		return err
	}

	c.Locals("userid", userID)
	err = c.Next()
	if c.Locals("data") == "" {
		return err
	}
	response.Data = c.Locals("data")
	return c.JSON(response)
}

// AuthenticateStream authenticates websocket connection
func AuthenticateStream(c *fiber.Ctx) error {

	if websocket.IsWebSocketUpgrade(c) {
		accessToken := c.Query("token")
		username := c.Query("username")

		if username == "" {
			return errors.HandleInvalidRequestError(c, "websocket_username", "empty username")
		}

		userID, err := helpers.ParseJWT(c, accessToken)
		if userID == "expired" {
			return errors.HandleInvalidRequestError(c, "Access Token", "expired")
		}

		if userID == "" {
			return err
		}
		c.Locals("userid", userID)
		c.Locals("username", username)
		return c.Next()
	}

	return errors.HandleInternalError(c, "websocket_upgrade", fiber.ErrUpgradeRequired.Error())
}

// SocialMiddleware preps for social service
func SocialMiddleware(c *fiber.Ctx) error {

	requestID := c.Query("requestid")

	if requestID == "" || requestID == c.Locals("userid").(string) {
		return errors.HandleBadRequestError(c, "Request ID", "invalid")
	}

	reqUsername, err := helpers.GetUsernameByID(requestID)

	if err != nil {
		if err == gocql.ErrNotFound {
			return errors.HandleBadRequestError(c, "UserID", "invalid")
		}
		return errors.HandleInternalError(c, "users_private", "ScyllaDB: "+err.Error())
	}

	c.Locals("requestid", requestID)
	c.Locals("reqUsername", reqUsername)
	fmt.Println(requestID, reqUsername)

	return c.Next()
}
