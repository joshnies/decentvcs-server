package middleware

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/lib/acl"
	"github.com/joshnies/decent-vcs/lib/auth"
	"github.com/joshnies/decent-vcs/models"
)

// Fiber middleware that ensures the user has access to the requested team.
// If `minRole` is nil, any role is allowed.
func HasTeamAccess(minRole models.Role) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userData := auth.GetUserDataFromContext(c)
		teamName := c.Params("team_name")

		// Check if user has access to team
		res, err := acl.HasTeamAccess(userData, teamName, minRole)
		if err != nil {
			fmt.Printf("[middleware.HasTeamAccess] Failed to determine team access: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
		if !res.HasAccess {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// Add team to user context for later use
		ctx := context.WithValue(c.UserContext(), models.ContextKeyTeam, res.Team)
		c.SetUserContext(ctx)

		return c.Next()
	}
}
