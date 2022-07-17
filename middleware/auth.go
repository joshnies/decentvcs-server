package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/constants"
	"github.com/joshnies/decent-vcs/models"
	"github.com/stytchauth/stytch-go/v5/stytch"
	"go.mongodb.org/mongo-driver/bson"
)

// Middleware that validates the Stytch session.
func IsAuthenticated(c *fiber.Ctx) error {
	sessionToken := c.Get(constants.SessionTokenHeader)
	if sessionToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(map[string]string{
			"error": "Unauthorized",
		})
	}

	// Authenticate the request using the session cookie
	res, err := config.StytchClient.Sessions.Authenticate(&stytch.SessionsAuthenticateParams{
		SessionToken: sessionToken,
		// SessionDurationMinutes: 0, // uncomment to reset session duration to * minutes from now
	})
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(map[string]string{
			"error": "Unauthorized",
		})
	}

	// Get user data
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": res.User.UserID}).Decode(&userData); err != nil {
		fmt.Printf("[middleware.IsAuthenticated] Failed to get user data: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
			"error": "Internal server error",
		})
	}

	// Add user data to context for later use
	userCtx := context.WithValue(c.UserContext(), models.ContextKeyUser, userData)
	c.SetUserContext(userCtx)

	return c.Next()
}
