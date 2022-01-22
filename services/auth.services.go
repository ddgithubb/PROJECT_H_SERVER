package services

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/schemas"

	"github.com/go-redis/redis/v8"
	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func Test(c *fiber.Ctx) error {

	req := new(schemas.LoginSchema)

	if err := c.BodyParser(req); err != nil {
		return errors.HandleBadJsonError(c)
	}

	if err := global.Validator.Struct(req); err != nil {
		return errors.HandleValidatorError(c, err)
	}

	temp := time.Now()
	fmt.Println(temp.Unix())
	fmt.Println(temp.UnixMilli())

	return helpers.OKResponse(c)
}

// Register users
func Register(c *fiber.Ctx) error {

	req := new(schemas.RegisterSchema)

	if err := c.BodyParser(req); err != nil {
		return errors.HandleBadJsonError(c)
	}

	if err := global.Validator.Struct(req); err != nil {
		return errors.HandleValidatorError(c, err)
	}

	validUser := validUsername.MatchString(req.Username)

	if !validUser {
		return errors.HandleBadRequestError(c, "Username", "regex")
	}

	var existCount int

	err := global.Session.Query(`
		SELECT count(*) FROM users_public WHERE username = ? LIMIT 1;`,
		req.Username,
	).WithContext(global.Context).Scan(&existCount)

	if err != nil {
		return errors.HandleInternalError(c, "users_public", "ScyllaDB: "+err.Error())
	}

	if existCount != 0 {
		return errors.HandleInvalidRequestError(c, "Username", "exists")
	}

	err = global.Session.Query(`
		SELECT count(*) FROM users WHERE email = ? LIMIT 1;`,
		req.Email,
	).WithContext(global.Context).Scan(&existCount)

	if err != nil {
		return errors.HandleInternalError(c, "users", "ScyllaDB: "+err.Error())
	}

	if existCount != 0 {
		return errors.HandleInvalidRequestError(c, "Email", "exists")
	}

	code, err := helpers.RandomTokenString(3)

	if err != nil {
		return errors.HandleInternalError(c, "code", "hex token error")
	}

	password, err := helpers.DecryptRequestPasswordHash(c, req.EncPasswordHash)
	if len(password) == 0 {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return errors.HandleInternalError(c, "password", "hashing error")
	}

	code = strings.ToUpper(code)
	req.Email = strings.ToLower(req.Email)

	query := map[string]interface{}{
		"code":         code,
		"username":     req.Username,
		"passwordhash": passwordHash,
	}

	redisError := false

	_, err = global.RedisClient.Pipelined(global.Context, func(pipe redis.Pipeliner) error {
		err = pipe.HSet(global.Context, "verifying:"+req.Email, query).Err()
		if err != nil {
			redisError = true
			return errors.HandleInternalError(c, "set_verifying", "Redis: "+err.Error())
		}
		err = pipe.Expire(global.Context, "verifying:"+req.Email, time.Hour*24).Err()
		if err != nil {
			redisError = true
			return errors.HandleInternalError(c, "expire_verifying", "Redis: "+err.Error())
		}
		return nil
	})

	if err != nil {
		return errors.HandleInternalError(c, "pipeline", "Redis: "+err.Error())
	}

	if redisError {
		return err
	}

	helpers.SendVerifEmail(c, req.Email, code)

	return helpers.OKResponse(c)
}

// ResendVerification resends verification email
func ResendVerification(c *fiber.Ctx) error {

	req := new(schemas.EmailSchema)

	if err := c.BodyParser(req); err != nil {
		return errors.HandleBadJsonError(c)
	}

	if err := global.Validator.Struct(req); err != nil {
		return errors.HandleValidatorError(c, err)
	}

	req.Email = strings.ToLower(req.Email)

	code, err := global.RedisClient.HGet(global.Context, "verifying:"+req.Email, "code").Result()
	if err != nil {
		if err == redis.Nil {
			return errors.HandleBadRequestError(c, "Email", "invalid")
		}
		return errors.HandleInternalError(c, "get_verifying", "Redis: "+err.Error())
	}

	helpers.SendVerifEmail(c, req.Email, code)

	return helpers.OKResponse(c)
}

// VerifyEmail verifies email
func VerifyEmail(c *fiber.Ctx) error {

	req := new(schemas.VerifyEmailSchema)

	if err := c.BodyParser(req); err != nil {
		return errors.HandleBadJsonError(c)
	}

	if err := global.Validator.Struct(req); err != nil {
		return errors.HandleValidatorError(c, err)
	}

	req.Email = strings.ToLower(req.Email)

	res, err := global.RedisClient.HGetAll(global.Context, "verifying:"+req.Email).Result()
	if err != nil {
		if err == redis.Nil {
			return errors.HandleBadRequestError(c, "Email", "invalid")
		}
		return errors.HandleInternalError(c, "getall_verifying", "Redis: "+err.Error())
	}

	if res["code"] != req.Code {
		return errors.HandleInvalidRequestError(c, "Code", "invalid")
	}

	userID := gocql.TimeUUID()

	applied, err := global.Session.Query(`
		INSERT INTO users_public (username,user_id)
		VALUES(?,?) 
		IF NOT EXISTS;`,
		res["username"],
		userID,
	).WithContext(global.Context).MapScanCAS(make(map[string]interface{}))

	if err != nil {
		return errors.HandleInternalError(c, "users_public", "ScyllaDB: "+err.Error())
	}
	if !applied {
		return errors.HandleBadRequestError(c, "Username", "exists")
	}

	err = global.Session.Query(`
		INSERT INTO users (email,user_id,password_hash,created)
		VALUES(?,?,?,?);`,
		req.Email,
		userID,
		res["passwordhash"],
		time.Now().UTC(),
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "users", "ScyllaDB: "+err.Error())
	}

	err = global.Session.Query(`
		INSERT INTO users_private (user_id,username,statement)
		VALUES(?,?,?);`,
		userID,
		res["username"],
		gocql.UnsetValue,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "users_private", "ScyllaDB: "+err.Error())
	}

	return helpers.OKResponse(c)
}

// Login log users in
func Login(c *fiber.Ctx) error {

	req := new(schemas.LoginSchema)

	if err := c.BodyParser(req); err != nil {
		return errors.HandleBadJsonError(c)
	}

	if err := global.Validator.Struct(req); err != nil {
		return errors.HandleValidatorError(c, err)
	}

	req.Email = strings.ToLower(req.Email)

	mainResult := make(map[string]interface{})

	err := global.Session.Query(`
		SELECT * FROM users WHERE email = ? LIMIT 1;`,
		req.Email,
	).WithContext(global.Context).MapScan(mainResult)

	if err != nil {
		if err == gocql.ErrNotFound {
			return errors.HandleInvalidRequestError(c, "Email", "invalid")
		}
		return errors.HandleInternalError(c, "users", "ScyllaDB: "+err.Error())
	}

	userID := mainResult["user_id"].(gocql.UUID).String()

	password, err := helpers.DecryptRequestPasswordHash(c, req.EncPasswordHash)
	if len(password) == 0 {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(mainResult["password_hash"].(string)), password)
	if err != nil {
		return errors.HandleInvalidRequestError(c, "Password", "invalid")
	}

	err = global.Session.Query(`
		INSERT INTO user_devices (user_id,created,device_token,active)
		VALUES(?,?,?,?) 
		IF NOT EXISTS;`,
		userID,
		time.Now().UTC(),
		req.DeviceToken,
		true,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_devices", "ScyllaDB: "+err.Error())
	}

	initUserInfo, err := helpers.GetUserInfo(c, userID)
	if initUserInfo.UserID == "" {
		return err
	}

	sessionID, err := helpers.RandomTokenString(20)
	if sessionID == "" {
		return err
	}

	c.Response().Header.Add("x-session-id", sessionID)

	if err = helpers.GenerateAndRefreshTokens(c, userID, sessionID, false); err != nil {
		return err
	}

	return c.JSON(initUserInfo)
}
