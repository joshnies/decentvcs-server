package middleware

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/decentvcs/server/config"
	"github.com/decentvcs/server/constants"
	"github.com/decentvcs/server/lib/acl"
	"github.com/decentvcs/server/lib/auth"
	"github.com/decentvcs/server/lib/team_lib"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
	"github.com/stytchauth/stytch-go/v5/stytch"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Middleware that validates the Stytch session.
func IsAuthenticated(c *fiber.Ctx) error {
	sessionToken := c.Get(constants.SessionTokenHeader)
	if sessionToken == "" {
		fmt.Println("[middleware.IsAuthenticated] No session token provided")
		return c.Status(fiber.StatusUnauthorized).JSON(map[string]string{
			"error": "Unauthorized",
		})
	}

	// Authenticate the request using the session cookie
	res, err := config.StytchClient.Sessions.Authenticate(&stytch.SessionsAuthenticateParams{
		SessionToken: sessionToken,
	})
	if err != nil {
		fmt.Printf("[middleware.IsAuthenticated] Failed to authenticate session: %v\n", err)
		fmt.Printf("[middleware.IsAuthenticated] Session token: %s\n", sessionToken)
		return c.Status(fiber.StatusUnauthorized).JSON(map[string]string{
			"error": "Unauthorized",
		})
	}

	stytchUser := res.User

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Get or create user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": stytchUser.UserID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// User data doesn't exist, so create it
			userData = models.UserData{
				ID:        primitive.NewObjectID(),
				CreatedAt: time.Now(),
				UserID:    stytchUser.UserID,
				Roles:     []models.RoleObject{},
			}
			if _, err := config.MI.DB.Collection("user_data").InsertOne(ctx, userData); err != nil {
				fmt.Printf("[middleware.IsAuthenticated] Error creating user data for user with ID \"%s\": %v\n", stytchUser.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// Create default team
			_, err := team_lib.CreateDefault(stytchUser.UserID, stytchUser.Emails[0].Email)
			if err != nil {
				fmt.Printf("[middleware.IsAuthenticated] Error creating default team for user with ID \"%s\": %v\n", stytchUser.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		} else {
			// Unhandled error occurred
			fmt.Printf("[middleware.IsAuthenticated] Error getting user data: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
				"error": "Internal server error",
			})
		}
	}

	// Add data to context for later use
	newUserCtx := context.WithValue(c.UserContext(), models.ContextKeyStytchUser, stytchUser)
	newUserCtx = context.WithValue(newUserCtx, models.ContextKeyUserData, userData)
	c.SetUserContext(newUserCtx)

	return c.Next()
}

// Fiber middleware that ensures the user has access to the requested team.
// If `minRole` is nil, any role is allowed.
//
// Assumes that `IsAuthenticated` was included as middleware BEFORE this one.
func HasTeamAccess(minRole models.Role) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userData := auth.GetUserDataFromContext(c)
		if userData == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(map[string]string{
				"error": "Unauthorized",
			})
		}

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
			fmt.Println("[middleware.HasTeamAccess] No access to team \"%s\"", teamName)
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
