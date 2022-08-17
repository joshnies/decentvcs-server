package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
	"github.com/joshnies/decent-vcs/models"
)

func RouteRoot(router fiber.Router) {
	router.Post("/session", controllers.CreateOrRefreshSession)
	router.Delete("/session", middleware.IsAuthenticated, controllers.RevokeSession)
	router.Post("/:team_name/invite", middleware.IsAuthenticated, middleware.HasTeamAccess(models.RoleAdmin), controllers.InviteToTeam)
	router.Get("/:team_name/projects", middleware.IsAuthenticated, middleware.HasTeamAccess(models.RoleNone), controllers.GetManyProjects)
	router.Delete("/:team_name/backdrop", middleware.IsAuthenticated, middleware.HasTeamAccess(models.RoleAdmin), controllers.DeleteTeamBackdrop)
}
