package routes

import (
	"PROJECT_H_server/services"

	"github.com/gofiber/fiber/v2"
)

func authRoutes(api fiber.Router) {
	oauth := api.Group("auth")
	oauth.Post("/test", services.Test)
	oauth.Post("/register", services.Register)
	oauth.Post("/resend-verification", services.ResendVerification)
	oauth.Post("/verify-email", services.VerifyEmail)
	oauth.Post("/login", services.Login)

	// router.HandleFunc("/forgot-password", )
	// router.HandleFunc("/revoke-token", )
	// router.HandleFunc("/reset-password", )
	// router.HandleFunc("/validate-reset-token", )
}
