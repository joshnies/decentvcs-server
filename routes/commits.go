package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/controllers"
)

func RouteCommits(router fiber.Router) {
	router.Get("/", controllers.GetManyCommits)
	router.Post("/", controllers.CreateCommit)
	router.Get("/:id", controllers.GetOneCommit)
}
