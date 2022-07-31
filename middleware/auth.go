package middleware

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/constants"
	"github.com/joshnies/decent-vcs/lib/team_lib"
	"github.com/joshnies/decent-vcs/models"
	"github.com/stytchauth/stytch-go/v5/stytch"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get or create user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": res.User.UserID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Create user data
			userData = models.UserData{
				ID:        primitive.NewObjectID(),
				CreatedAt: time.Now(),
				UserID:    res.User.UserID,
				Roles:     []models.RoleObject{},
			}
			if _, err := config.MI.DB.Collection("user_data").InsertOne(ctx, userData); err != nil {
				fmt.Printf("Error creating user data for user with ID \"%s\": %v\n", res.User.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// Create new default team
			_, err := team_lib.CreateDefault(res.User.UserID, res.User.Emails[0].Email)
			if err != nil {
				fmt.Printf("Error creating default team for user with ID \"%s\": %v\n", res.User.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		} else {
			fmt.Printf("Error fetching user data for user with ID \"%s\": %v\n", res.User.UserID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	// Add user data to context for later use
	userCtx := context.WithValue(c.UserContext(), models.ContextKeyUser, userData)
	c.SetUserContext(userCtx)

	return c.Next()
}
