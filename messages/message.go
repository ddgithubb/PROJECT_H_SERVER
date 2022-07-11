package messages

import (
	"PROJECT_H_server/errors"
	"time"

	"github.com/aidarkhanov/nanoid/v2"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
)

const VALID_NANOID_CHAR = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func send_payload(c *fiber.Ctx, op op_type, targetID string, atomic bool, i interface{}) error {

	signature, err := nanoid.GenerateString(VALID_NANOID_CHAR, 4)
	if err != nil {
		log_err("Nanoid generation: " + err.Error())
		return err
	}

	json_string, err := jsoniter.Marshal(construct_ws_message(op, c.Locals("userid").(string), targetID, time.Now().UnixMilli(), signature, atomic, i))
	if err != nil {
		return errors.HandleInternalError(c, "jsoniter_marshal", err.Error())
	}

	err = api_write_message(100, json_string, c.Locals("sessionid").(string), c.Locals("userid").(string), targetID)
	if err != nil {
		return errors.HandleInternalError(c, "api_write_message", err.Error())
	}

	return nil
}
