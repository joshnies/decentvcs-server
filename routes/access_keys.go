package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/gofiber/fiber/v2"
)

func RouteAccessKeys(router fiber.Router) {
	router.Post("/", controllers.CreateAccessKey)
	router.Delete("/", controllers.DeleteAccessKey)
}
