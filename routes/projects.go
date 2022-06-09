package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs-api/controllers"
	"github.com/joshnies/decent-vcs-api/lib/acl"
	"github.com/joshnies/decent-vcs-api/middleware"
)

func RouteProjects(router fiber.Router) {
	router.Post("/", controllers.CreateProject)
	router.Get("/blob/:oa/:pname", controllers.GetOneProjectByBlob)
	router.Get("/:pid", controllers.GetOneProject, middleware.HasProjectAccess)
	router.Post("/:pid", controllers.UpdateOneProject, middleware.HasProjectAccess)
	router.Delete("/:pid", controllers.DeleteOneProject, middleware.HasProjectAccessWithRole(acl.RoleOwner))
	router.Post("/:pid/invite", controllers.InviteManyUsers, middleware.HasProjectAccessWithRole(acl.RoleAdmin))
}
