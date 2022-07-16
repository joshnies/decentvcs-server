package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
	"github.com/joshnies/decent-vcs/models"
)

func RouteTeams(router fiber.Router) {
	router.Use(middleware.IsAuthenticated)

	router.Get("/", controllers.GetManyTeams)
	router.Post("/", controllers.CreateTeam)
	router.Get("/:team_name", controllers.GetOneTeam)
	router.Post("/:team_name", middleware.HasTeamAccess(models.RoleAdmin), controllers.UpdateTeam)
	router.Delete("/:team_name", middleware.HasTeamAccess(models.RoleOwner), controllers.DeleteTeam)
}
