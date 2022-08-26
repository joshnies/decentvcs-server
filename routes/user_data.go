package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/decentvcs/server/middleware"
	"github.com/gofiber/fiber/v2"
)

func RouteUserData(router fiber.Router) {
	router.Use(middleware.IsAuthenticated)

	router.Get("/me", controllers.GetUserData)
	router.Put("/me", controllers.UpdateUserData)
}
