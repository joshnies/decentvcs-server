package routes

import (
	"github.com/decentvcs/server/controllers"
	"github.com/decentvcs/server/middleware"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
)

func RouteBranches(router fiber.Router) {
	router.Use(middleware.IsAuthenticated, middleware.HasTeamAccess(models.RoleNone))

	router.Get("/", controllers.GetManyBranches)
	router.Post("/", controllers.CreateBranch)
	router.Get("/default", controllers.GetDefaultBranch)
	router.Get("/:branch_name", controllers.GetOneBranch)
	router.Put("/:branch_name", controllers.UpdateBranch)
	router.Delete("/:branch_name", controllers.SoftDeleteOneBranch)
	router.Post("/:branch_name/commit", controllers.CreateCommit)
	router.Delete("/:branch_name/commits", controllers.DeleteManyCommitsInBranch)

	RouteLocks(router.Group("/:branch_name/locks"))
}
