package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/team_lib"
	"github.com/joshnies/decent-vcs/models"
	"github.com/stytchauth/stytch-go/v5/stytch"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Authenticate Stytch session token.
// If `SessionToken` is provided, the existing session will be refreshed instead of creating a new one.
func Authenticate(c *fiber.Ctx) error {
	// Validate request body
	var body models.AuthenticateRequest
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	var userID string
	var email string
	var sessionToken string
	if body.TokenType == "magic_link" {
		// Authenticate magic link token
		stytchres, err := config.StytchClient.MagicLinks.Authenticate(&stytch.MagicLinksAuthenticateParams{
			Token:                  body.Token,
			SessionToken:           body.SessionToken,
			SessionDurationMinutes: config.I.Stytch.SessionDurationMinutes,
			Attributes: stytch.Attributes{
				IPAddress: c.IP(),
			},
		})
		if err != nil {
			return err
		}

		userID = stytchres.UserID
		email = stytchres.User.Emails[0].Email
		sessionToken = stytchres.SessionToken
	} else if body.TokenType == "oauth" {
		// Authenticate OAuth token
		stytchres, err := config.StytchClient.OAuth.Authenticate(&stytch.OAuthAuthenticateParams{
			Token:                  body.Token,
			SessionToken:           body.SessionToken,
			SessionDurationMinutes: config.I.Stytch.SessionDurationMinutes,
		})
		if err != nil {
			return err
		}

		userID = stytchres.UserID
		sessionToken = stytchres.SessionToken

		// Get user email from Stytch
		// This is required since Stytch doesn't return the user's email after oauth
		stytchUserRes, err := config.StytchClient.Users.Get(stytchres.UserID)
		if err != nil {
			return err
		}

		email = stytchUserRes.Emails[0].Email
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get or create user data from database.
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": userID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Create user data
			userData = models.UserData{
				ID:        primitive.NewObjectID(),
				CreatedAt: time.Now(),
				UserID:    userID,
				Roles:     []models.RoleObject{},
			}
			if _, err := config.MI.DB.Collection("user_data").InsertOne(ctx, userData); err != nil {
				fmt.Printf("Error creating user data while authenticating user with ID \"%s\": %v\n", userID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		} else {
			fmt.Printf("Error fetching user data while authenticating user with ID \"%s\": %v\n", userID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	// Create the user's default team if it doesn't exist.
	if userData.DefaultTeamID.IsZero() {
		// Create new default team
		_, err := team_lib.CreateDefault(userID, email)
		if err != nil {
			fmt.Printf("Error creating default team during authentication: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	// Return response
	res := models.AuthenticateResponse{
		SessionToken: sessionToken,
	}

	return c.JSON(res)
}

// Revoke the session.
func RevokeSession(c *fiber.Ctx) error {
	sessionToken := c.Get("X-Session-Token") // already validated in the `ValidateAuth` middleware

	_, err := config.StytchClient.Sessions.Revoke(&stytch.SessionsRevokeParams{
		SessionToken: sessionToken,
	})

	return err
}
