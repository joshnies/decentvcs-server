package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/gofiber/fiber/v2"
)

func RouteLocks(router fiber.Router) {
	router.Post("/", controllers.Lock)
	router.Delete("/", controllers.Unlock)
}
