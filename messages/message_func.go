package messages

import (
	"github.com/gofiber/fiber/v2"
)

func FriendRequest(c *fiber.Ctx, targetID, destinationUsername string) error {

	err := send_payload(c, 200, targetID, false, friend_request_data{
		OriginUsername: c.Locals("username").(string),
		TargetUsername: destinationUsername,
	})

	return err

}

func FriendAccepted(c *fiber.Ctx, targetID string, chainID string, created int64) error {

	err := send_payload(c, 201, targetID, false, friend_accept_data{
		ChainID: chainID,
		Created: created,
	})

	return err
}

func RelationRemove(c *fiber.Ctx, targetID string) error {

	err := send_payload(c, 202, targetID, false, nil)

	return err
}

func SendMessage(c *fiber.Ctx, targetID string, chainID string, messageID string, created int64, expires int64, typeCode int, display string, duration int64) error {

	err := send_payload(c, 300, targetID, true, message_data{
		ChainID:   chainID,
		MessageID: messageID,
		Created:   created,
		Expires:   expires,
		Type:      typeCode,
		Display:   display,
		Duration:  duration,
	})

	return err
}
