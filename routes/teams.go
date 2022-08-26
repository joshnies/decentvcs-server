package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/decentvcs/server/middleware"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
)

func RouteTeams(router fiber.Router) {
	router.Use(middleware.IsAuthenticated)

	router.Get("/", controllers.GetManyTeams)
	router.Post("/", controllers.CreateTeam)
	router.Get("/:team_name", controllers.GetOneTeam)
	router.Put("/:team_name", middleware.HasTeamAccess(models.RoleAdmin), controllers.UpdateTeam)
	router.Delete("/:team_name", middleware.HasTeamAccess(models.RoleOwner), controllers.DeleteTeam)
}
