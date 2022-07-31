package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/constants"
	"github.com/joshnies/decent-vcs/lib/auth"
	"github.com/joshnies/decent-vcs/lib/team_lib"
	"github.com/joshnies/decent-vcs/models"
	"github.com/stytchauth/stytch-go/v5/stytch"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Create or refresh a Stytch session.
// This is usually only needed to refresh an existing session, since the website handles all auth flows.
//
// To refresh an existing session, provide `SessionToken`.
func CreateOrRefreshSession(c *fiber.Ctx) error {
	// Validate request body
	var body models.AuthenticateRequest
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	var sessionToken string
	if body.TokenType == "magic_links" {
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

		sessionToken = stytchres.SessionToken
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid token type \"%s\"; must be either `magic_link` or `oauth`", body.TokenType),
		})
	}

	// Return response
	res := models.AuthenticateResponse{
		SessionToken: sessionToken,
	}

	return c.JSON(res)
}

// Revoke the session.
func RevokeSession(c *fiber.Ctx) error {
	sessionToken := c.Get(constants.SessionTokenHeader) // already validated in the `ValidateAuth` middleware

	_, err := config.StytchClient.Sessions.Revoke(&stytch.SessionsRevokeParams{
		SessionToken: sessionToken,
	})

	return err
}

// Initialize all required resources for a user.
// If already initialized, nothing happens and no error is returned.
func Init(c *fiber.Ctx) error {
	stytchUser := auth.GetStytchUserFromContext(c)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Get or create user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": stytchUser.UserID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Create user data
			userData = models.UserData{
				ID:        primitive.NewObjectID(),
				CreatedAt: time.Now(),
				UserID:    stytchUser.UserID,
				Roles:     []models.RoleObject{},
			}
			if _, err := config.MI.DB.Collection("user_data").InsertOne(ctx, userData); err != nil {
				fmt.Printf("Error creating user data for user with ID \"%s\": %v\n", stytchUser.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		} else {
			fmt.Printf("Error fetching user data for user with ID \"%s\": %v\n", stytchUser.UserID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	// Get or create default team
	if userData.DefaultTeamID.IsZero() {
		_, err := team_lib.CreateDefault(stytchUser.UserID, stytchUser.Emails[0].Email)
		if err != nil {
			fmt.Printf("Error creating default team for user with ID \"%s\": %v\n", stytchUser.UserID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
	})
}
