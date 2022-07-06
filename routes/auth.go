package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
)

func RouteAuth(router fiber.Router) {
	router.Post("/authenticate", controllers.Authenticate)
	router.Delete("/session", middleware.IsAuthenticated, controllers.RevokeSession)
}
