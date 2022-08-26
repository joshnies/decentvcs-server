package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/decentvcs/server/middleware"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
)

func RouteCommits(router fiber.Router) {
	router.Use(middleware.IsAuthenticated, middleware.HasTeamAccess(models.RoleNone))

	router.Get("/", controllers.GetManyCommits)
	router.Get("/:commit_index", controllers.GetOneCommit)
	router.Put("/:commit_index", controllers.UpdateCommit)
}
