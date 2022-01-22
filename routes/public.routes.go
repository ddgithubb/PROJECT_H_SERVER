package routes

import (
	"PROJECT_H_server/services"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
)

func publicRoutes(api fiber.Router) {
	public := api.Group("/public")
	public.Use(cache.New(cache.Config{
		Next: func(c *fiber.Ctx) bool {
			return c.Query("refresh") == "true"
		},
		Expiration:   5 * time.Minute,
		CacheControl: true,
	}))
	public.Get("/user", services.UserByUsername)
}
