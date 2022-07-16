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
	router.Get("/:project_name", middleware.HasTeamAccess, controllers.GetOneProject)
	router.Post("/:project_name", middleware.HasTeamAccess, controllers.UpdateOneProject)
	router.Delete("/:project_name", middleware.HasTeamAccessWithRole(models.RoleOwner), controllers.DeleteOneProject)
	router.Post("/:project_name/invite", middleware.HasTeamAccessWithRole(models.RoleAdmin), controllers.InviteManyUsers)
}
