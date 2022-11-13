package routes

import (
	"github.com/decentvcs/server/constants"
	"github.com/decentvcs/server/controllers"
	"github.com/decentvcs/server/middleware"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
)

func RouteAccessKeys(router fiber.Router) {
	router.Post(
		"/",
		middleware.HasAccessKeyScope(constants.ScopeTeamUpdateUsage),
		middleware.HasTeamAccess(models.RoleCollab),
		controllers.CreateAccessKey,
	)
	router.Delete(
		"/",
		middleware.HasAccessKeyScope(constants.ScopeTeamUpdateUsage),
		middleware.HasTeamAccess(models.RoleCollab),
		controllers.DeleteAccessKey,
	)
}
