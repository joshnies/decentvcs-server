package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/lib/auth"
)

// Get one user from Stytch.
func GetOneStytchUser(c *fiber.Ctx) error {
	// Get Stytch user ID
	userID := c.Params("uid")

	// Get user
	user, err := auth.GetStytchUserByID(userID)
	if err != nil {
		return err
	}

	// Return user
	return c.JSON(user)
}
