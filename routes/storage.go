package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/quanta-api/controllers"
)

func RouteStorage(router fiber.Router) {
	router.Post("/presign/many", controllers.PresignManyGET)
	router.Post("/presign/:method", controllers.PresignOne)
	router.Post("/multipart/complete", controllers.CompleteMultipartUpload)
}
