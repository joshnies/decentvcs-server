package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
	"github.com/joshnies/decent-vcs/middleware"
)

func RouteStorage(router fiber.Router) {
	router.Use(middleware.IsAuthenticated, middleware.HasTeamAccess)

	router.Post("/presign/many", controllers.PresignManyGET)
	router.Post("/presign/:method", controllers.PresignOne)
	router.Post("/multipart/complete", controllers.CompleteMultipartUpload)
	router.Delete("/unused", controllers.DeleteUnusedStorageObjects)
}
