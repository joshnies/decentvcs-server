package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/decentvcs/server/middleware"
	"github.com/gofiber/fiber/v2"
)

func RouteBilling(router fiber.Router) {
	router.Use(middleware.IsAuthenticated)

	router.Post("/subscriptions", controllers.GetOrCreateBillingSubscription)
}
