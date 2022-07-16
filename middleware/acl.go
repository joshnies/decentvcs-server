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
		// Get URL params
		teamName := c.Params("team_name")

		// Get user ID
		userID, err := auth.GetUserID(c)
		if err != nil {
			return err
		}

		// Check if user has access to team
		res, err := acl.HasTeamAccess(userID, teamName, minRole)
		if err != nil {
			fmt.Println(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
		if !res.HasAccess {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// Add user to context for later use
		ctx := context.WithValue(c.UserContext(), models.ContextKeyTeam, res.Team)
		c.SetUserContext(ctx)

		return c.Next()
	}
}
