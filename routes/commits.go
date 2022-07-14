package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
)

func RouteCommits(router fiber.Router) {
	router.Use(middleware.IsAuthenticated, middleware.HasTeamAccess)

	router.Get("/", controllers.GetManyCommits)
	router.Post("/", controllers.CreateOneCommit)
	router.Get("/index/:idx", controllers.GetOneCommitByIndex)
	router.Get("/:cid", controllers.GetOneCommitByID)
	router.Post("/:cid", controllers.UpdateOneCommit)
}
