package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
	"github.com/joshnies/decent-vcs/models"
)

func RouteProjects(router fiber.Router) {
	router.Use(middleware.IsAuthenticated)

	router.Post("/", middleware.HasTeamAccess(models.RoleNone), controllers.CreateProject)
	router.Get("/:project_name", middleware.HasTeamAccess(models.RoleNone), controllers.GetOneProject)
	router.Post("/:project_name", middleware.HasTeamAccess(models.RoleNone), controllers.UpdateOneProject)
	router.Delete("/:project_name", middleware.HasTeamAccess(models.RoleOwner), controllers.DeleteOneProject)
	router.Post("/:project_name/invite", middleware.HasTeamAccess(models.RoleAdmin), controllers.InviteManyUsers)
}
