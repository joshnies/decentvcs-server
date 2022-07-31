package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/models"
	"github.com/stytchauth/stytch-go/v5/stytch"
)

// Get Stytch user from user context.
func GetStytchUserFromContext(c *fiber.Ctx) stytch.User {
	return c.UserContext().Value(models.ContextKeyStytchUser).(stytch.User)
}

// Get user data from user context.
func GetUserDataFromContext(c *fiber.Ctx) models.UserData {
	return c.UserContext().Value(models.ContextKeyUserData).(models.UserData)
}

// Returns the user's ID from the session.
// @deprecated - Use `GetUserFromContext` instead.
func GetUserID(c *fiber.Ctx) (string, error) {
	userVal := c.UserContext().Value(models.ContextKeyUserData)
	if userVal == nil {
		return "", fiber.ErrUnauthorized
	}

	return userVal.(stytch.User).UserID, nil
}

// Get a user from Stytch by ID.
func GetStytchUserByID(userID string) (*stytch.UsersGetResponse, error) {
	res, err := config.StytchClient.Users.Get(userID)
	if err != nil {
		return nil, err
	}

	return res, nil
}
