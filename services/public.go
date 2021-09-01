package services

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/schemas"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

// UserByUsername gets user by username
func UserByUsername(c *fiber.Ctx) error {

	username := c.Query("username")

	if username == "" || len(username) > 30 {
		return errors.HandleInvalidRequestError(c, "Username", "invalid")
	}

	res := make(map[string]interface{})

	err := global.Session.Query(`
		SELECT * FROM users_public WHERE username = ? LIMIT 1;`,
		username,
	).WithContext(global.Context).MapScan(res)

	if err != nil {
		if err == gocql.ErrNotFound {
			return errors.HandleInvalidRequestError(c, "Username", "invalid")
		}
		return errors.HandleInternalError(c, "users_public", "ScyllaDB: "+err.Error())
	}

	return c.JSON(schemas.PublicUserSchema{
		Username:  res["username"].(string),
		UserID:    res["user_id"].(gocql.UUID).String(),
		Statement: res["statement"].(string),
	})
}
