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
	router.Get("/", middleware.HasTeamAccess(models.RoleNone), controllers.GetOneProject)
	router.Put("/", middleware.HasTeamAccess(models.RoleNone), controllers.UpdateProject)
	router.Delete("/", middleware.HasTeamAccess(models.RoleOwner), controllers.DeleteOneProject)
	router.Post("/transfer", middleware.HasTeamAccess(models.RoleOwner), controllers.TransferProjectOwnership)
}
