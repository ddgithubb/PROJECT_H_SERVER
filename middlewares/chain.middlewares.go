package middlewares

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/helpers"

	"github.com/gofiber/fiber/v2"
)

// ChainRequestMiddleware checks the validity of the request
func ChainRequestMiddleware(c *fiber.Ctx) error {

	chainID, err := helpers.ParseParamChainUUID(c)
	if err != nil {
		return err
	}

	users, err := helpers.GetChainsUsers(c, chainID)
	if err != nil {
		return err
	}

	if len(users) != 2 {
		// TODO: GROUPS
		return errors.HandleBadRequestError(c, "ChainID", "invalid")
	}

	if users[0] == c.Locals("userid").(string) {
		c.Locals("requestid", users[1])
	} else if users[1] == c.Locals("userid").(string) {
		c.Locals("requestid", users[0])
	} else {
		return errors.HandleBadRequestError(c, "ChainID", "unauthorized")
	}

	c.Locals("chainid", chainID)

	return c.Next()
}
