package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/controllers"
)

func RouteBranches(router fiber.Router) {
	router.Get("/", controllers.GetManyBranches)
	router.Post("/", controllers.CreateBranch)
	router.Get("/:bid", controllers.GetOneBranch)
	router.Get("/:bid/commits", controllers.GetManyCommitsForBranch)
}
