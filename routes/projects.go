package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
	"github.com/joshnies/decent-vcs/models"
)

func RouteProjects(router fiber.Router) {
	router.Post("/", controllers.CreateProject)
	router.Get("/blob/:oa/:pname", controllers.GetOneProjectByBlob)
	router.Get("/:pid", controllers.GetOneProject, middleware.HasProjectAccess)
	router.Post("/:pid", controllers.UpdateOneProject, middleware.HasProjectAccess)
	router.Delete("/:pid", controllers.DeleteOneProject, middleware.HasProjectAccessWithRole(models.RoleOwner))
	router.Post("/:pid/invite", controllers.InviteManyUsers, middleware.HasProjectAccessWithRole(models.RoleAdmin))
}
