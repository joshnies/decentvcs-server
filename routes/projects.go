package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
	"github.com/joshnies/decent-vcs/models"
)

func RouteProjects(router fiber.Router) {
	router.Use(middleware.IsAuthenticated)

	router.Post("/", controllers.CreateProject)
	router.Get("/blob/:oa/:pname", controllers.GetOneProjectByBlob)
	router.Get("/:pid", middleware.HasProjectAccess, controllers.GetOneProject)
	router.Post("/:pid", middleware.HasProjectAccess, controllers.UpdateOneProject)
	router.Delete("/:pid", middleware.HasProjectAccessWithRole(models.RoleOwner), controllers.DeleteOneProject)
	router.Post("/:pid/invite", middleware.HasProjectAccessWithRole(models.RoleAdmin), controllers.InviteManyUsers)
}
