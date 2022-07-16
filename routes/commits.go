package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
	"github.com/joshnies/decent-vcs/models"
)

func RouteCommits(router fiber.Router) {
	router.Use(middleware.IsAuthenticated, middleware.HasTeamAccess(models.RoleNone))

	router.Get("/", controllers.GetManyCommits)
	router.Post("/", controllers.CreateOneCommit)
	router.Get("/index/:commit_index", controllers.GetOneCommitByIndex)
	router.Get("/:commit_id", controllers.GetOneCommitByID)
	router.Post("/:commit_id", controllers.UpdateOneCommit)
}
