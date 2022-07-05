package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/teams"
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

	// Get or create the user's default team from the database
	var team models.Team
	if err := config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"owner_user_id": userID}).Decode(&team); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Create default team
			team, err = teams.CreateDefault(userID, email)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		} else {
			fmt.Printf("Error fetching default team while authenticating user with ID \"%s\": %v\n", userID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	// Get or create user data from database.
	// NOTE: User data is fetched only to ensure it's there.
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": userID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Create user data
			userData = models.UserData{
				ID:            primitive.NewObjectID(),
				CreatedAt:     time.Now().Unix(),
				UserID:        userID,
				Roles:         []models.RoleObject{},
				DefaultTeamID: team.ID,
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
