package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
)

func RouteBranches(router fiber.Router) {
	router.Use(middleware.IsAuthenticated, middleware.HasTeamAccess)

	router.Get("/", controllers.GetManyBranches)
	router.Post("/", controllers.CreateBranch)
	router.Get("/default", controllers.GetDefaultBranch)
	router.Get("/:bid", controllers.GetOneBranch)
	router.Post("/:bid", controllers.UpdateOneBranch)
	router.Delete("/:bid", controllers.DeleteOneBranch)
	router.Get("/:bid/commits", controllers.GetManyCommitsForBranch)
	router.Delete("/:bid/commits", controllers.DeleteManyCommitsInBranch)

	RouteLocks(router.Group("/:bid/locks"))
}
