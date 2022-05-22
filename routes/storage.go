package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/controllers"
)

func RouteStorage(router fiber.Router) {
	router.Post("/presign/:method", controllers.CreatePresignedURLs)
	router.Post("/multipart/complete", controllers.CompleteMultipartUpload)
}
