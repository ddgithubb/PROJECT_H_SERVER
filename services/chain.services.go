package services

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/schemas"
	"strconv"
	"time"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

// GetChain gets a segment of chain for specified relation
func GetChain(c *fiber.Ctx) error {

	chainID := c.Locals("chainid").(gocql.UUID)

	request, err := strconv.ParseInt(c.Query("requestTime"), 10, 64)
	if err != nil {
		return errors.HandleInternalError(c, "parse_request_time", err.Error())
	}
	requestTime := time.UnixMilli(request)
	asc := c.Query("asc")
	desc := c.Query("desc")
	limit, err := strconv.ParseInt(c.Query("limit", "50"), 10, 64)
	if err != nil {
		return errors.HandleInternalError(c, "parse_limit", err.Error())
	}

	if asc != "true" && desc != "true" {
		newChain, err := helpers.GetChain(chainID, requestTime, false, true, limit)
		if err != nil {
			return errors.HandleInternalError(c, "helpers_get_chain", err.Error())
		}

		return c.JSON(newChain)
	} else {

		ascChain := []schemas.MessageSchema{}
		descChain := []schemas.MessageSchema{}

		if asc == "true" {
			ascChain, err = helpers.GetChain(chainID, requestTime, true, false, limit)
			if err != nil {
				return errors.HandleInternalError(c, "helpers_get_chain", err.Error())
			}
		}

		if desc == "true" {
			descChain, err = helpers.GetChain(chainID, requestTime, false, false, limit)
			if err != nil {
				return errors.HandleInternalError(c, "helpers_get_chain", err.Error())
			}
		}

		return c.JSON(append(descChain, ascChain...)) //Not a big performance hit as descChain is often greater than ascChain
	}
}