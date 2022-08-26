package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/decentvcs/server/middleware"
	"github.com/gofiber/fiber/v2"
)

func RouteStytch(router fiber.Router) {
	router.Use(middleware.IsAuthenticated)

	router.Get("/users/:user_id", controllers.GetOneStytchUser)
}
