package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/models"
	"github.com/stytchauth/stytch-go/v5/stytch"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Authenticate Stytch session token.
func Authenticate(c *fiber.Ctx) error {
	// Validate request body
	var body models.AuthenticateRequest
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	// Authenticate magic link token
	//
	// If `SessionToken` is provided, the existing session will be refreshed instead of creating a new one
	stytchres, err := config.StytchClient.MagicLinks.Authenticate(&stytch.MagicLinksAuthenticateParams{
		Token:                  body.Token,
		SessionToken:           body.SessionToken,
		SessionDurationMinutes: config.I.Stytch.SessionDurationMinutes,
		// TODO: Add IP matching
		// Options:    stytch.Options{IPMatchRequired: true},
		// Attributes: stytch.Attributes{IPAddress: "10.0.0.0"},
	})
	if err != nil {
		return err
	}

	// Return response
	res := models.AuthenticateResponse{
		SessionToken: stytchres.SessionToken,
	}

	// Get or create user data from database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": stytchres.UserID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Create user data
			userData = models.UserData{
				ID:        primitive.NewObjectID(),
				CreatedAt: time.Now().Unix(),
				UserID:    stytchres.UserID,
				Roles:     []models.RoleObject{},
			}
			if _, err := config.MI.DB.Collection("user_data").InsertOne(ctx, userData); err != nil {
				fmt.Printf("Error creating user data while authenticating user with ID \"%s\": %v\n", stytchres.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			return c.JSON(res)
		}

		fmt.Printf("Error fetching user data while authenticating user with ID \"%s\": %v\n", stytchres.UserID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
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
