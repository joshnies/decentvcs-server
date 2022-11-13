package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/decentvcs/server/middleware"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
)

func RouteAccessKeys(router fiber.Router) {
	router.Use(middleware.IsAuthenticated)

	router.Post(
		"/",
		middleware.HasTeamAccess(models.RoleCollab),
		controllers.CreateAccessKey,
	)
	router.Delete(
		"/",
		middleware.HasTeamAccess(models.RoleCollab),
		controllers.DeleteAccessKey,
	)
}
