package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
)

func RouteRoot(router fiber.Router) {
	router.Post("/session", controllers.CreateOrRefreshSession)
	router.Delete("/session", middleware.IsAuthenticated, controllers.RevokeSession)
}
