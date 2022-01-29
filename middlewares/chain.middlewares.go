package middlewares

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

// ChainRequestMiddleware checks the validity of the request
func ChainRequestMiddleware(c *fiber.Ctx) error {

	chainID, err := helpers.ParseChainUUID(c)
	if err != nil {
		return err
	}

	userRelationsResult := make(map[string]interface{})

	err = global.Session.Query(`
		SELECT * FROM user_relations WHERE user_id = ? AND created = ? LIMIT 1;`,
		c.Locals("userid").(string),
		chainID.Time(),
	).WithContext(global.Context).MapScan(userRelationsResult)
	if err != nil {
		if err == gocql.ErrNotFound {
			return errors.HandleInternalError(c, "users_relations", "ScyllaDB: "+err.Error())
		}
		return errors.HandleInternalError(c, "users_relations", "ScyllaDB: "+err.Error())
	}

	if chainID.String() != userRelationsResult["chain_id"].(gocql.UUID).String() {
		return errors.HandleBadRequestError(c, "ChainID", "invalid")
	}

	c.Locals("chainid", chainID)
	c.Locals("requestid", userRelationsResult["relation_id"].(gocql.UUID).String())
	return c.Next()
}
