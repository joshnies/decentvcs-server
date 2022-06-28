package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/stytchauth/stytch-go/v5/stytch"
)

// Middleware that validates the Stytch session.
func ValidateAuth(c *fiber.Ctx) error {
	sessionToken := c.Get("X-Session-Token")
	if sessionToken == "" {
		c.Status(401).JSON(map[string]string{
			"error": "Unauthorized",
		})
		return nil
	}

	// Authenticate the request using the session cookie
	res, err := config.StytchClient.Sessions.Authenticate(&stytch.SessionsAuthenticateParams{
		SessionToken: sessionToken,
		// SessionDurationMinutes: 0, // uncomment to reset session duration to * minutes from now
	})
	if err != nil {
		c.Status(401).JSON(map[string]string{
			"error": "Unauthorized",
		})
		return nil
	}

	// Add user to context for later use
	ctx := context.WithValue(c.UserContext(), "user", res.User)
	c.SetUserContext(ctx)

	return c.Next()
}
