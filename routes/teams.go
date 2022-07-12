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
	router.Get("/:tid", controllers.GetOneTeam)
	router.Post("/", controllers.CreateTeam)
	router.Post("/:tid", middleware.HasTeamAccessWithRole(models.RoleAdmin), controllers.UpdateTeam)
	router.Delete("/:tid", middleware.HasTeamAccessWithRole(models.RoleOwner), controllers.DeleteTeam)
}
