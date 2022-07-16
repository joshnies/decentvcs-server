package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/lib/acl"
	"github.com/joshnies/decent-vcs/lib/auth"
	"github.com/joshnies/decent-vcs/models"
)

// Fiber middleware that ensures the user has access to the requested team as any role.
func HasTeamAccess(c *fiber.Ctx) error {
	// Get URL params
	teamName := c.Params("team_name")

	// Get user ID
	userID, err := auth.GetUserID(c)
	if err != nil {
		return err
	}

	// Check if user has access to team
	hasAccess, err := acl.HasTeamAccess(userID, teamName, models.RoleNone)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	if !hasAccess {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	return c.Next()
}

// Fiber middleware that ensures the user has access to the requested team with a role greater or equal to the
// `minRole`.
func HasTeamAccessWithRole(minRole models.Role) func(*fiber.Ctx) error {
	// Return middleware function
	return func(c *fiber.Ctx) error {
		// Get URL params
		teamName := c.Params("team_name")

		// Get user ID
		userId, err := auth.GetUserID(c)
		if err != nil {
			return err
		}

		// Check if user has role for team
		userRole, err := acl.GetTeamRole(userId, teamName)
		if err != nil {
			fmt.Println(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		userRoleLvl, err := acl.GetRoleLevel(userRole)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		minRoleLvl, err := acl.GetRoleLevel(minRole)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		if userRoleLvl < minRoleLvl {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden",
			})
		}

		return c.Next()
	}
}
