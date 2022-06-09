package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs-api/lib/acl"
	"github.com/joshnies/decent-vcs-api/lib/auth"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Fiber middleware that validates the following:
//
// - Project ID (`pid`) is valid
//
// - User has access to the requested project as any role
//
func HasProjectAccess(c *fiber.Ctx) error {
	// Get project ID
	pid := c.Params("pid")
	_, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID; must be an ObjectID hexadecimal",
		})
	}

	// Check if user has access to project
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	hasAccess, err := acl.HasProjectAccess(userId, pid)
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

// Fiber middleware that validates the following:
//
// - Project ID (`pid`) is valid
//
// - User has access to the requested project with a role greater or equal to the `minRole`
//
func HasProjectAccessWithRole(minRole acl.Role) func(*fiber.Ctx) error {
	// Return middleware function
	return func(c *fiber.Ctx) error {
		// Get project ID
		pid := c.Params("pid")
		_, err := primitive.ObjectIDFromHex(pid)
		if err != nil {
			fmt.Println(err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": "Invalid project ID; must be an ObjectID hexadecimal",
			})
		}

		// Check if user has role for project
		userId, err := auth.GetUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		userRole, err := acl.GetProjectRole(userId, pid)
		if err != nil {
			fmt.Println(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
		if userRole < minRole {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		return c.Next()
	}
}
