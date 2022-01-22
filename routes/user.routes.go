package routes

import (
	"PROJECT_H_server/middlewares"
	"PROJECT_H_server/services"

	"github.com/gofiber/fiber/v2"
)

func userRoutes(api fiber.Router) {
	user := api.Group("/user")
	user.Use(middlewares.Authenticate)

	user.Get("/initialize", services.Initialize)

	user.Post("/request", services.Request)
	user.Post("/remove-relation", services.RemoveRelation)
	user.Post("/accept", services.Accept)

	chainRoutes(user)
}

func chainRoutes(api fiber.Router) {
	chain := api.Group("/chain")

	userChainRoutes(chain)
}

func userChainRoutes(api fiber.Router) {
	userChain := api.Group("/:chainID")
	userChain.Post("/send-audio", services.SendAudio)
	userChain.Get("/get-chain", services.GetChain)
}
