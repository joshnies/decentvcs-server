package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
)

func RouteLocks(router fiber.Router) {
	router.Post("/", controllers.Lock)
	router.Delete("/", controllers.Unlock)
}
