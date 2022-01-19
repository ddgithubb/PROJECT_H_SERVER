package routes

import (
	"PROJECT_H_server/middlewares"
	"PROJECT_H_server/services"

	"github.com/gofiber/fiber/v2"
)

func socialRoutes(api fiber.Router) {
	resources := api.Group("social")
	resources.Use(middlewares.Authenticate)
	resources.Use(middlewares.SocialMiddleware)
	resources.Post("/request", services.Request)
	resources.Post("/remove-relation", services.RemoveRelation)
	resources.Post("/accept", services.Accept)
}
