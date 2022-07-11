package routes

import (
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/middlewares"
	"PROJECT_H_server/services"

	"github.com/gofiber/fiber/v2"
)

func userRoutes(api fiber.Router) {
	user := api.Group("/user")
	user.Use(middlewares.Authenticate)

	user.Get("/authorize", helpers.OKResponse)
	user.Get("/initialize", services.Initialize)
	
	relationRoutes(user)
	chainRoutes(user)
}