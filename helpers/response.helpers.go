package helpers

import (
	"PROJECT_H_server/schemas"

	"github.com/gofiber/fiber/v2"
)

// OKResponse sends a successful request/response
func OKResponse(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(schemas.ErrorResponse{
		Error: false,
	})
}
