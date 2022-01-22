package helpers

import (
	"PROJECT_H_server/config"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"strconv"
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
	return c.SendStatus(fiber.StatusOK)
}

// MilisecondsToTime converts milliseconds since epoch to golang time object
func MilisecondsToTime(milli int64) time.Time {
	return time.UnixMilli(milli)
}

// ParseStringToInt parses string to int
func ParseStringToInt(str string) (int64, error) {
	return strconv.ParseInt(str, 10, 64)
}
