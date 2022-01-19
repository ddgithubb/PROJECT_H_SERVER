package routes

import (
	"PROJECT_H_server/config"
	"PROJECT_H_server/middlewares"
	"PROJECT_H_server/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
)

// SetRoutes sets all routes of server
func SetRoutes(app *fiber.App) {
	api := app.Group(config.Config.Version)
	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))

	app.Use("/stream", middlewares.AuthenticateStream, websocket.New(services.Stream))

	authRoutes(api)
	socialRoutes(api)
	resourceRoutes(api)
	formRoutes(api)
	publicRoutes(api)
}
