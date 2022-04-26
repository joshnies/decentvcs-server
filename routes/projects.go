package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/controllers"
)

func RouteProjects(router fiber.Router) {
	router.Get("/", controllers.GetManyProjects)
	router.Post("/", controllers.CreateProject)
	router.Get("/:id", controllers.GetOneProject)
	router.Get("/:id", controllers.GetAccessGrant)
}
