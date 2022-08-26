package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/decentvcs/server/middleware"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
)

func RouteProjects(router fiber.Router) {
	router.Use(middleware.IsAuthenticated)

	router.Post("/", middleware.HasTeamAccess(models.RoleNone), controllers.CreateProject)
	router.Get("/", middleware.HasTeamAccess(models.RoleNone), controllers.GetOneProject)
	router.Put("/", middleware.HasTeamAccess(models.RoleNone), controllers.UpdateProject)
	router.Delete("/", middleware.HasTeamAccess(models.RoleOwner), controllers.DeleteOneProject)
	router.Post("/transfer", middleware.HasTeamAccess(models.RoleOwner), controllers.TransferProjectOwnership)
}
