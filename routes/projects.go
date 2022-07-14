package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
	"github.com/joshnies/decent-vcs/models"
)

func RouteProjects(router fiber.Router) {
	router.Use(middleware.IsAuthenticated)

	router.Post("/", middleware.HasTeamAccess, controllers.CreateProject)
	router.Get("/:pid", middleware.HasTeamAccess, controllers.GetOneProject)
	router.Post("/:pid", middleware.HasTeamAccess, controllers.UpdateOneProject)
	router.Delete("/:pid", middleware.HasTeamAccessWithRole(models.RoleOwner), controllers.DeleteOneProject)
	router.Post("/:pid/invite", middleware.HasTeamAccessWithRole(models.RoleAdmin), controllers.InviteManyUsers)
}
