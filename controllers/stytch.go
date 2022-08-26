package controllers

import (
	"github.com/decentvcs/server/lib/auth"
	"github.com/gofiber/fiber/v2"
)

// Get one user from Stytch.
func GetOneStytchUser(c *fiber.Ctx) error {
	// Get Stytch user ID
	userID := c.Params("user_id")

	// Get user
	user, err := auth.GetStytchUserByID(userID)
	if err != nil {
		return err
	}

	// Return user
	return c.JSON(user)
}
