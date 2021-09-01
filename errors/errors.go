package errors

import (
	"PROJECT_H_server/global"
	"PROJECT_H_server/schemas"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// HandleFatalError handles global error
func HandleFatalError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

// HandleBasicError handles basic error and logs
func HandleBasicError(err error) bool {
	if err != nil {
		global.Logger.Println(err)
		return true
	}
	return false
}

// HandleInternalError handles internal errors (things that should never happen in normal circumstances)
func HandleInternalError(c *fiber.Ctx, problem string, err string) error {
	global.Logger.Println("ip: " + c.IP() + "; Problem: " + problem + "; Error: " + err)
	return c.Status(fiber.StatusInternalServerError).JSON(schemas.ErrorResponse{
		Error: true,
	})
}

// HandleBadRequestError handles bad request errors (client error that is harmless to server and state)
func HandleBadRequestError(c *fiber.Ctx, errorType string, problem string) error {
	return c.Status(fiber.StatusBadRequest).JSON(schemas.ErrorResponse{
		Error:   true,
		Type:    errorType,
		Problem: problem,
	})
}

// HandleInvalidRequestError handles invalid request errors (expected errors)
func HandleInvalidRequestError(c *fiber.Ctx, errorType string, problem string) error {
	return c.Status(fiber.StatusAccepted).JSON(schemas.ErrorResponse{
		Error:   true,
		Type:    errorType,
		Problem: problem,
	})
}

// HandleValidatorError handles errors when validating request
func HandleValidatorError(c *fiber.Ctx, err error) error {
	validatorErr := err.(validator.ValidationErrors)[0]
	return HandleBadRequestError(c, validatorErr.StructField(), validatorErr.Tag())
}

// HandleBadJsonError handles json request parser errors
func HandleBadJsonError(c *fiber.Ctx) error {
	return HandleBadRequestError(c, "JSON body", "invalid")
}

// HandleWebsocketError handles internal errors from websocket
func HandleWebsocketError(c *websocket.Conn, problem string, err string) {
	global.Logger.Println("ip: " + c.RemoteAddr().String() + "; Problem: " + problem + "; Error: " + err)
}
