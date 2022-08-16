package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/constants"
	"github.com/joshnies/decent-vcs/lib/acl"
	"github.com/joshnies/decent-vcs/lib/auth"
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
	})
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(map[string]string{
			"error": "Unauthorized",
		})
	}

	// Add Stytch user to context for later use
	userCtx := context.WithValue(c.UserContext(), models.ContextKeyStytchUser, res.User)
	c.SetUserContext(userCtx)

	return c.Next()
}

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

// Middleware that adds user data to context.
func IncludeUserData(c *fiber.Ctx) error {
	stytchUser := auth.GetStytchUserFromContext(c)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": stytchUser.UserID}).Decode(&userData); err != nil {
		fmt.Printf("[middleware.IncludeUserData] Error getting user data: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
			"error": "Internal server error",
		})
	}

	// Add user data to context for later use
	userDataCtx := context.WithValue(c.UserContext(), models.ContextKeyUserData, userData)
	c.SetUserContext(userDataCtx)

	return c.Next()
}
