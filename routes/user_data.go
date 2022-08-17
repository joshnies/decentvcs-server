package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
)

func RouteUserData(router fiber.Router) {
	router.Use(middleware.IsAuthenticated)

	router.Get("/me", controllers.GetUserData)
	router.Put("/me", controllers.UpdateUserData)
}
