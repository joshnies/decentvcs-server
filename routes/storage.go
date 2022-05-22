package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/controllers"
)

func RouteStorage(router fiber.Router) {
	router.Post("/presign/:method", controllers.PresignOne)
	router.Post("/presign/many/:method", controllers.PresignMany)
	router.Post("/multipart/complete", controllers.CompleteMultipartUpload)
}