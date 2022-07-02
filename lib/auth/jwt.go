package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/models"
	"github.com/stytchauth/stytch-go/v5/stytch"
)

// Returns the user's ID from the session.
func GetUserID(c *fiber.Ctx) (string, error) {
	userVal := c.UserContext().Value(models.ContextKeyUser)
	if userVal == nil {
		return "", fiber.ErrUnauthorized
	}

	return userVal.(stytch.User).UserID, nil
}
