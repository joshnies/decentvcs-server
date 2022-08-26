package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/decentvcs/server/middleware"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
)

func RouteRoot(router fiber.Router) {
	router.Get("/health", controllers.Ping)
	router.Post("/session", controllers.CreateOrRefreshSession)
	router.Delete("/session", middleware.IsAuthenticated, controllers.RevokeSession)
	router.Post("/:team_name/invite", middleware.IsAuthenticated, middleware.HasTeamAccess(models.RoleAdmin), controllers.InviteToTeam)
	router.Get("/:team_name/projects", middleware.IsAuthenticated, middleware.HasTeamAccess(models.RoleNone), controllers.GetManyProjects)
	router.Delete("/:team_name/backdrop", middleware.IsAuthenticated, middleware.HasTeamAccess(models.RoleAdmin), controllers.DeleteTeamBackdrop)
}
