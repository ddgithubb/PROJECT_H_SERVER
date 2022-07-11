package helpers

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

// GetUsernameByID gets only the username column by id
func GetUsernameByID(c *fiber.Ctx, id string) (string, error) {

	reqUsername := ""

	err := global.Session.Query(`
		SELECT username FROM users_private WHERE user_id = ? LIMIT 1;`,
		id,
	).WithContext(global.Context).Scan(&reqUsername)

	if err != nil {
		if err == gocql.ErrNotFound {
			return "", errors.HandleBadRequestError(c, "UserID", "invalid")
		}
		return "", errors.HandleInternalError(c, "users_private", "ScyllaDB: "+err.Error())
	}

	return reqUsername, err
}

// CheckUser checks user by id
func CheckUser(id string) (bool, error) {

	var existCount int

	err := global.Session.Query(`
		SELECT count(*) FROM users_private WHERE user_id = ? LIMIT 1;`,
		id,
	).WithContext(global.Context).Scan(&existCount)

	if existCount == 0 {
		return false, err
	}
	return true, err
}
