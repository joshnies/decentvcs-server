package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/controllers"
)

func RouteCommits(router fiber.Router) {
	router.Get("/", controllers.GetManyCommits)
	router.Post("/", controllers.CreateOneCommit)
	router.Get("/index/:idx", controllers.GetOneCommitByIndex)
	router.Get("/:cid", controllers.GetOneCommitByID)
	router.Post("/:cid", controllers.UpdateOneCommit)
}
