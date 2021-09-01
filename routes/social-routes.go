package routes

import (
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/services"

	"github.com/gofiber/fiber/v2"
)

func socialRoutes(api fiber.Router) {
	resources := api.Group("social")
	resources.Use(helpers.Authenticate)
	resources.Use(helpers.SocialMiddleware)
	resources.Post("/request", services.Request)
	resources.Post("/remove-relation", services.RemoveRelation)
	resources.Post("/accept", services.Accept)
}
