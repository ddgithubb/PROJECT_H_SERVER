package helpers

import (
	"PROJECT_H_server/config"
	"PROJECT_H_server/schemas"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RandomTokenString generates random hex token
func RandomTokenString(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// EmailSender sends an email to 1 account
func EmailSender(to string, message string) error {
	plainAuth := smtp.PlainAuth("", config.Config.SMTP.User, config.Config.SMTP.Password, config.Config.SMTP.Host)
	err := smtp.SendMail(config.Config.SMTP.Host+":"+fmt.Sprint(config.Config.SMTP.Port), plainAuth, config.Config.SMTP.User, []string{to}, []byte(message))
	if err != nil {
		return err
	}
	return nil
}

// OKResponse sends a successful request/response
func OKResponse(c *fiber.Ctx) error {
	return c.JSON(schemas.Message{
		Message: "OK",
	})
}

// MilisecondsToTime converts milliseconds since epoch to golang time object
func MilisecondsToTime(milli int64) time.Time {
	return time.UnixMilli(milli)
}

// ReturnData populates local data and returns nil
func ReturnData(c *fiber.Ctx, data interface{}) error {
	c.Locals("data", data)
	return nil
}

// ReturnOKData returns ok message in local data and returns nil
func ReturnOKData(c *fiber.Ctx) error {
	c.Locals("data", "OK")
	return nil
}
