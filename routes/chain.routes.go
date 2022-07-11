package routes

import (
	"PROJECT_H_server/middlewares"
	"PROJECT_H_server/services"

	"github.com/gofiber/fiber/v2"
)

func chainRoutes(api fiber.Router) {
	chain := api.Group("/chain")

	userChainRoutes(chain)
}

func userChainRoutes(api fiber.Router) {
	userChain := api.Group("/:chainID", middlewares.ChainRequestMiddleware)
	userChain.Get("/get-chain", services.GetChain)
	userChain.Post("/audio", services.AddAudioMessage)
	userChain.Get("/audio/:messageID", services.GetAudioMessage)
}
