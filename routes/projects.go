package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/controllers"
)

func RouteProjects(router fiber.Router) {
	router.Get("/", controllers.GetManyProjects)
	router.Post("/", controllers.CreateProject)
	router.Get("/blob/:oa/:pname", controllers.GetOneProjectByBlob)
	router.Get("/:pid", controllers.GetOneProject)
	router.Post("/:pid", controllers.UpdateOneProject)
	router.Get("/:pid/access_grant", controllers.GetAccessGrant)
}
