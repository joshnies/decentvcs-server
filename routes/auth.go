package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
)

func RouteAuth(router fiber.Router) {
	router.Post("/authenticate", controllers.Authenticate)
}
