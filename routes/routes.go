package routes

import (
	"PROJECT_H_server/config"
	"PROJECT_H_server/socket"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
)

// SetRoutes sets all routes of server
func SetRoutes(app *fiber.App) {
	api := app.Group("/api/" + config.Config.Version)
	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))

	app.Use("/socket", socket.InitializeSocket, websocket.New(socket.ClientSocket))

	authRoutes(api)
	userRoutes(api)
	publicRoutes(api)
}
