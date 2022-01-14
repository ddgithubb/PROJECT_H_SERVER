package routes

import (
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/services"

	"github.com/gofiber/fiber/v2"
)

func resourceRoutes(api fiber.Router) {
	resources := api.Group("resources")
	resources.Use(helpers.Authenticate)
	resources.Post("/authenticate", services.Authenticate)
	resources.Post("/get-chain", services.GetChain)
}

func formRoutes(api fiber.Router) {
	forms := api.Group("/forms")
	forms.Use(helpers.AuthenticateForm)
	forms.Post("/send-audio", services.SendAudio)
}
