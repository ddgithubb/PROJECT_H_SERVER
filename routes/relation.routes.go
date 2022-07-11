package routes

import (
	"PROJECT_H_server/services"

	"github.com/gofiber/fiber/v2"
)

func relationRoutes(api fiber.Router) {
	relation := api.Group("/relation")

	api.Get("/relation", services.GetAllRelations)
	userRelationRoutes(relation)
}

func userRelationRoutes(api fiber.Router) {
	userRelation := api.Group("/:relationID")
	userRelation.Post("/request", services.RequestRelation)
	userRelation.Post("/remove", services.RemoveRelation)
	userRelation.Post("/accept", services.AcceptRelation)
}
