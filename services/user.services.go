package services

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/schemas"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

// Initialize retrieves user data for initial app state
func Initialize(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)

	userResult := make(map[string]interface{})

	err := global.Session.Query(`
		SELECT * FROM users_private WHERE user_id = ? LIMIT 1;`,
		userID,
	).WithContext(global.Context).MapScan(userResult)

	if err != nil {
		if err == gocql.ErrNotFound {
			return errors.HandleInternalError(c, "users_private", "ScyllaDB: "+err.Error())
		}
		return errors.HandleInternalError(c, "users_private", "ScyllaDB: "+err.Error())
	}

	initUserInfo := schemas.UserInfoSchema{
		UserID:   userID,
		Username: userResult["username"].(string),
	}

	err = helpers.RelationsMapper(&initUserInfo.Relations, userID)
	if err != nil {
		return errors.HandleInternalError(c, "user_relations", err.Error())
	}

	if statement, ok := userResult["statement"].(string); ok {
		initUserInfo.Statement = statement
	}

	return c.JSON(initUserInfo)
}
