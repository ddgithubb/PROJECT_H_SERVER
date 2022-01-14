package services

import (
	"PROJECT_H_server/helpers"

	"github.com/gofiber/fiber/v2"
)

// Authenticate authenticates users with a refresh token
func Authenticate(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)

	initUserInfo, err := helpers.GetUserInfo(c, userID)
	if initUserInfo.UserID == "" {
		return err
	}

	return helpers.ReturnData(c, initUserInfo)
}
