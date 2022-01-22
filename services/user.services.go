package services

import (
	"PROJECT_H_server/helpers"

	"github.com/gofiber/fiber/v2"
)

// Initialize retrieves user data for initial app state
func Initialize(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)

	initUserInfo, err := helpers.GetUserInfo(c, userID)
	if initUserInfo.UserID == "" {
		return err
	}

	return c.JSON(initUserInfo)
}
