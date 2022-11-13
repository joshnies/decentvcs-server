package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	// Get access key from header
	accessKey := c.Get(constants.AccessKeyHeader)
	if accessKey != "" {
		// Authenticate with access key
		fmt.Println("[middleware.IsAuthenticated] Authenticating with access key") // DEBUG
		return authenticateWithAccessKey(c, accessKey)
	}

	// Get session token from header for Stytch auth
	fmt.Println("[middleware.IsAuthenticated] Authenticating with Stytch") // DEBUG
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
			if _, err = config.MI.DB.Collection("user_data").InsertOne(ctx, userData); err != nil {
				fmt.Printf("[middleware.IsAuthenticated] Error creating user data for user with ID \"%s\": %v\n", stytchUser.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// Create default team
			_, userData, err = team_lib.CreateDefault(stytchUser.UserID, stytchUser.Emails[0].Email)
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
		// Get user data & team from context
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
			fmt.Printf("[middleware.HasTeamAccess] No access to team \"%s\"\n", teamName)
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

func authenticateWithAccessKey(c *fiber.Ctx, accessKeyIDHex string) error {
	// Get access key from database
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	accessKeyID, err := primitive.ObjectIDFromHex(accessKeyIDHex)
	if err != nil {
		fmt.Println("[middleware.IsAuthenticated] Unauthorized: Invalid access key ID")
		return c.Status(fiber.StatusUnauthorized).JSON(map[string]string{
			"error": "Unauthorized",
		})
	}

	var dbAccessKey models.AccessKey
	if err := config.MI.DB.Collection("access_keys").FindOne(ctx, bson.M{"_id": accessKeyID}).Decode(&dbAccessKey); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Access key doesn't exist
			fmt.Println("[middleware.IsAuthenticated] Unauthorized: Access key doesn't exist")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		} else {
			// Unhandled error occurred
			fmt.Printf("[middleware.authenticateWithAccessKey] Error getting access key: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
				"error": "Internal server error",
			})
		}
	}

	// Check if expired
	if dbAccessKey.ExpiresAt.Before(time.Now()) {
		// Delete expired access key from database
		if _, err := config.MI.DB.Collection("access_keys").DeleteOne(ctx, bson.M{"id": accessKeyID}); err != nil {
			fmt.Printf("[middleware.authenticateWithAccessKey] Error deleting expired access key: %v\n", err)
		}

		fmt.Println("[middleware.IsAuthenticated] Unauthorized: Access key expired")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": dbAccessKey.UserID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// User data doesn't exist
			fmt.Printf("[middleware.authenticateWithAccessKey] Access key \"%s\" references a user that doesn't exist\n", accessKeyIDHex)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		} else {
			// Unhandled error occurred
			fmt.Printf("[middleware.authenticateWithAccessKey] Error getting user data: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
				"error": "Internal server error",
			})
		}
	}

	// Add data to context for later use
	newUserCtx := context.WithValue(c.UserContext(), models.ContextKeyUserData, userData)
	newUserCtx = context.WithValue(newUserCtx, models.ContextKeyAccessKey, dbAccessKey)
	c.SetUserContext(newUserCtx)

	return c.Next()
}

// Fiber middleware that ensures that the access key has the required scope to access the requested resource.
func HasAccessKeyScope(scope string) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		if c.UserContext().Value(models.ContextKeyAccessKey) == nil {
			// No access key, was authenticated with Stytch
			return c.Next()
		}

		// Get access key from context
		accessKey := c.UserContext().Value(models.ContextKeyAccessKey).(*models.AccessKey)
		if accessKey == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(map[string]string{
				"error": "Unauthorized",
			})
		}

		// Check if access key has scope
		for _, s := range strings.Split(accessKey.Scope, " ") {
			if s == scope {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusUnauthorized).JSON(map[string]string{
			"error": "Unauthorized",
		})
	}
}
