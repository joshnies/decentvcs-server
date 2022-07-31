package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
)

func RouteStytch(router fiber.Router) {
	router.Use(middleware.IsAuthenticated, middleware.IncludeUserData)

	router.Get("/users/:user_id", controllers.GetOneStytchUser)
}
