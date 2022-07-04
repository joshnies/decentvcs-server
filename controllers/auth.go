package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
		Attributes: stytch.Attributes{
			IPAddress: c.IP(),
			// UserAgent: c.Get("User-Agent"),
		},
		// Options: stytch.Options{IPMatchRequired: true},
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get or create the user's default team from the database
	var team models.Team
	if err := config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"owner_user_id": stytchres.UserID}).Decode(&team); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Create default team
			email := stytchres.User.Emails[0].Email
			emailUser := strings.Split(email, "@")[0]

			// Check if there's a team already with that name
			alreadyExists := true
			var existingTeam models.Team
			if err := config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"name": emailUser}).Decode(&existingTeam); err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					alreadyExists = false
				} else {
					fmt.Printf("Error searching for existing team with name \"%s\": %v\n", emailUser, err)
				}
			}

			teamName := emailUser
			if alreadyExists {
				teamName += "-" + strings.Replace(strings.Replace(stytchres.UserID, "user-", "", 1), "test-", "", 1)
			}

			team = models.Team{
				ID:          primitive.NewObjectID(),
				CreatedAt:   time.Now().Unix(),
				OwnerUserID: stytchres.UserID,
				Name:        teamName,
			}
			if _, err := config.MI.DB.Collection("teams").InsertOne(ctx, team); err != nil {
				fmt.Printf("Error creating default team while authenticating user with ID \"%s\": %v\n", stytchres.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		} else {
			fmt.Printf("Error fetching default team while authenticating user with ID \"%s\": %v\n", stytchres.UserID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	// Get or create user data from database.
	// NOTE: User data is fetched only to ensure it's there.
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": stytchres.UserID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Create user data
			userData = models.UserData{
				ID:            primitive.NewObjectID(),
				CreatedAt:     time.Now().Unix(),
				UserID:        stytchres.UserID,
				Roles:         []models.RoleObject{},
				DefaultTeamID: team.ID,
			}
			if _, err := config.MI.DB.Collection("user_data").InsertOne(ctx, userData); err != nil {
				fmt.Printf("Error creating user data while authenticating user with ID \"%s\": %v\n", stytchres.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		} else {
			fmt.Printf("Error fetching user data while authenticating user with ID \"%s\": %v\n", stytchres.UserID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	// Return response
	res := models.AuthenticateResponse{
		SessionToken: stytchres.SessionToken,
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
