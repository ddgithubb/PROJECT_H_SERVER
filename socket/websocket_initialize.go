package socket

import (
	"PROJECT_H_server/errors"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// InitializeSocket initializes websocket connection
func InitializeSocket(c *fiber.Ctx) error {

	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}

	return errors.HandleInternalError(c, "websocket_upgrade", fiber.ErrUpgradeRequired.Error())
}
