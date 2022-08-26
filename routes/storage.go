package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/decentvcs/server/middleware"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
)

func RouteStorage(router fiber.Router) {
	router.Use(middleware.IsAuthenticated, middleware.HasTeamAccess(models.RoleNone))

	router.Post("/presign/many", controllers.PresignManyGET)
	router.Post("/presign/:method", controllers.PresignOne)
	router.Post("/multipart/complete", controllers.CompleteMultipartUpload)
	router.Delete("/unused", controllers.DeleteUnusedStorageObjects)
}
