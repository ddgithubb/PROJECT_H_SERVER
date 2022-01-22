package errors

import (
	"PROJECT_H_server/global"
	"PROJECT_H_server/schemas"
	Errors "errors"
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

// HandleComplexError handles complex errors and logs
func HandleComplexError(problem string, err string) error {
	global.Logger.Println("Problem: " + problem + "; Error: " + err)
	return Errors.New("Problem: " + problem + "; Error: " + err)
}

// HandleInternalError handles internal errors (things that should never happen in normal circumstances)
func HandleInternalError(c *fiber.Ctx, problem string, err string) error {
	global.Logger.Println("IP: " + c.IP() + "; Problem: " + problem + "; Error: " + err)
	return c.Status(fiber.StatusInternalServerError).JSON(schemas.ErrorResponse{
		Error: true,
	})
}

// HandleUnauthorizedError handles authorization error
func HandleUnauthorizedError(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(schemas.ErrorResponse{
		Error:       true,
		Problem:     "Authorization",
		Description: "unauthorized",
	})
}

// HandleBadRequestError handles bad request errors (client error that is harmless to server and state)
func HandleBadRequestError(c *fiber.Ctx, problem string, description string) error {
	return c.Status(fiber.StatusBadRequest).JSON(schemas.ErrorResponse{
		Error:       true,
		Problem:     problem,
		Description: description,
	})
}

// HandleInvalidRequestError handles invalid request errors (expected errors)
func HandleInvalidRequestError(c *fiber.Ctx, problem string, description string) error {
	return c.Status(fiber.StatusAccepted).JSON(schemas.ErrorResponse{
		Error:       true,
		Problem:     problem,
		Description: description,
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
