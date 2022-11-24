package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/decentvcs/server/config"
	"github.com/decentvcs/server/lib/auth"
	"github.com/decentvcs/server/lib/team_lib"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
	"github.com/sendgrid/sendgrid-go"
	sgmail "github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/stytchauth/stytch-go/v5/stytch"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Get one team.
func GetOneTeam(c *fiber.Ctx) error {
	// Return team from context
	team := c.UserContext().Value(models.ContextKeyTeam)

	if team == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Team not found",
		})
	}

	return c.JSON(team)
}

// Get many teams.
func GetManyTeams(c *fiber.Ctx) error {
	userData := auth.GetUserDataFromContext(c)

	// Get pagination query parameters
	var skip int64 = 0
	var limit int64 = 25

	if c.Query("skip") != "" {
		skip, _ = strconv.ParseInt(c.Query("skip"), 10, 64)
	}

	if c.Query("limit") != "" {
		limit, _ = strconv.ParseInt(c.Query("limit"), 10, 64)
	}

	// Build query
	query := bson.M{}
	mine := c.Query("mine") == "true"

	if mine {
		// Include only the user's teams
		query["$or"] = []bson.M{
			{"id": userData.DefaultTeamID},
		}

		for _, role := range userData.Roles {
			query["$or"] = append(query["$or"].([]bson.M), bson.M{"id": role.TeamID})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all teams (paginated)
	var teams []models.Team
	cur, err := config.MI.DB.Collection("teams").Find(ctx, bson.M{}, &options.FindOptions{
		Skip:  &skip,
		Limit: &limit,
	})
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Team not found",
		})
	}

	cur.All(ctx, &teams)

	if teams == nil {
		teams = []models.Team{} // Return empty array instead of null
	}

	return c.JSON(teams)
}

// Create a new team.
func CreateTeam(c *fiber.Ctx) error {
	// Get user ID
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse request body
	var reqBody models.CreateTeamRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request body
	if err := config.Validator.Struct(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if team name is unique
	var existingTeam models.Team
	err = config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"name": reqBody.Name}).Decode(&existingTeam)
	if err == nil || !errors.Is(err, mongo.ErrNoDocuments) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "A team with that name already exists",
		})
	}
	if err != nil {
		fmt.Printf("Error checking if team name is unique: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create team
	teamID := primitive.NewObjectID()
	team := models.Team{
		ID:        teamID,
		CreatedAt: time.Now(),
		Name:      reqBody.Name,
	}

	if _, err := config.MI.DB.Collection("teams").InsertOne(ctx, team); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create team",
		})
	}

	// Add team owner role to user
	if _, err := config.MI.DB.Collection("users").UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$push": bson.M{"roles": models.RoleObject{
		Role:   models.RoleOwner,
		TeamID: teamID,
	}}}); err != nil {
		fmt.Printf("Error adding team owner role to user: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(team)
}

// Update a team.
// Only team admins can update a team.
func UpdateTeam(c *fiber.Ctx) error {
	// Get team from context
	team := c.UserContext().Value(models.ContextKeyTeam).(*models.Team)

	// Parse request body
	var reqBody models.UpdateTeamRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request body
	if err := config.Validator.Struct(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if reqBody.Name != "" {
		// Check if team name is unique
		var existingTeam models.Team
		err := config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"name": reqBody.Name}).Decode(&existingTeam)
		if err == nil || !errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "A team with that name already exists",
			})
		}
		if err != nil {
			fmt.Printf("Error checking if team name is unique: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	updateData := bson.M{}

	if reqBody.Name != "" {
		updateData["name"] = reqBody.Name
	}

	// Update team
	if _, err := config.MI.DB.Collection("teams").UpdateOne(ctx, bson.M{"_id": team.ID}, bson.M{"$set": updateData}); err != nil {
		fmt.Printf("Error updating team: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	team.Name = reqBody.Name
	return c.JSON(team)
}

// Add to a team's usage metrics.
func UpdateTeamUsage(c *fiber.Ctx) error {
	// Get team from context
	team := c.UserContext().Value(models.ContextKeyTeam).(*models.Team)

	// Parse request body
	var reqBody models.UpdateTeamUsageRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request body
	if err := config.Validator.Struct(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Update team model locally (for response) and construct data for update query
	updateData := bson.M{}

	if reqBody.StorageUsedMB > 0 {
		team.StorageUsedMB = team.StorageUsedMB + reqBody.StorageUsedMB
		updateData["storage_used_mb"] = team.StorageUsedMB
	}

	if reqBody.BandwidthUsedMB > 0 {
		team.BandwidthUsedMB = team.BandwidthUsedMB + reqBody.BandwidthUsedMB
		updateData["bandwidth_used_mb"] = team.BandwidthUsedMB
	}

	// Update team
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := config.MI.DB.Collection("teams").UpdateOne(ctx, bson.M{"_id": team.ID}, bson.M{"$set": updateData}); err != nil {
		fmt.Printf("Error updating team: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(team)
}

// Delete a team.
// Only team owners can delete a team.
// A user's default team cannot be deleted.
func DeleteTeam(c *fiber.Ctx) error {
	// Get team from context
	team := c.UserContext().Value(models.ContextKeyTeam).(*models.Team)

	// Get user ID
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse request body
	var reqBody models.CreateTeamRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request body
	if err := config.Validator.Struct(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": userID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			fmt.Printf("User data not found for user with ID \"%s\"", userID)
		} else {
			fmt.Printf("Error getting user data: %v\n", err)
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Ensure deleting team is not the default for the user
	if team.ID.Hex() == userData.DefaultTeamID.Hex() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete default team",
		})
	}

	// Delete team
	if _, err := config.MI.DB.Collection("teams").DeleteOne(ctx, bson.M{"_id": team.ID}); err != nil {
		fmt.Printf("Error deleting team: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Team deleted successfully",
	})
}

// Invite many users to a team.
func InviteToTeam(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)

	// Parse request body
	var body models.InviteManyUsersDTO
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Limit email count to prevent request timeouts
	if len(body.Emails) > config.I.MaxInviteCount {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": fmt.Sprintf("Too many emails; limit: %d", config.I.MaxInviteCount),
		})
	}

	// Validate emails in request body
	for _, email := range body.Emails {
		if _, err := mail.ParseAddress(email); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": fmt.Sprintf("Invalid email: %s", email),
			})
		}
	}

	// Loop through emails
	for _, email := range body.Emails {
		// Check if user exists
		searchRes, err := config.StytchClient.Users.Search(&stytch.UsersSearchParams{
			Limit: 1,
			Query: &stytch.UsersSearchQuery{
				Operator: stytch.UserSearchOperatorAND,
				Operands: []json.Marshaler{
					stytch.UsersSearchQueryEmailAddressFilter{EmailAddresses: []string{email}},
				},
			},
		})
		if err != nil {
			fmt.Println(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		if len(searchRes.Results) == 0 {
			// User does not exist in auth provider, invite them via email
			inviteRes, err := config.StytchClient.MagicLinks.Email.Invite(&stytch.MagicLinksEmailInviteParams{
				Email:                   email,
				InviteMagicLinkURL:      config.I.Stytch.InviteRedirectURL,
				InviteExpirationMinutes: 1440, // 24 hours
				Attributes: stytch.Attributes{
					IPAddress: c.IP(),
					// UserAgent: c.Get("User-Agent"),
				},
			})
			if err != nil {
				fmt.Printf("Error sending invite to \"%s\" %v\n", email, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// Create user data in database with project role
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			preferredUsername := strings.Split(email, "@")[0]
			username, err := team_lib.GenerateUsername(preferredUsername)
			if err != nil {
				fmt.Printf("Error generating username: %v\n", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			userData := models.UserData{
				UserID:        inviteRes.UserID,
				Username:      username,
				DefaultTeamID: team.ID,
				Roles:         []models.RoleObject{},
			}
			if _, err := config.MI.DB.Collection("user_data").InsertOne(ctx, userData); err != nil {
				fmt.Printf("Error creating user data for new invited user with ID \"%s\": %v\n", inviteRes.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// Create default team in database
			// _, userData, err = team_lib.CreateDefault(inviteRes.UserID, email)
			// if err != nil {
			// 	fmt.Printf("Error creating default team for user with ID \"%s\": %v\n", inviteRes.UserID, err)
			// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			// 		"error": "Internal server error",
			// 	})
			// }
		} else {
			// User already exists in auth provider, get user data from database
			stytchUser := searchRes.Results[0]
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var userData models.UserData
			if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": stytchUser.UserID}).Decode(&userData); err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					fmt.Printf("User data not found while inviting existing user with ID \"%s\" to a project: %v\n", stytchUser.UserID, err)
				} else {
					fmt.Printf("Error getting user data while inviting existing user with ID \"%s\" to a project: %v\n", stytchUser.UserID, err)
				}

				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// Skip this user if they already have a role for the project
			skip := false
			for _, r := range userData.Roles {
				if r.TeamID.Hex() == team.ID.Hex() {
					skip = true
					if config.I.Debug {
						fmt.Printf("Skipped inviting user with ID \"%s\" to team \"%s\" since they already have a role for the team\n", team.Name, stytchUser.UserID)
					}
					break
				}
			}
			if skip {
				continue
			}

			// Add role to user data
			userData.Roles = append(userData.Roles, models.RoleObject{
				Role:   models.RoleCollab,
				TeamID: team.ID,
			})
			if _, err := config.MI.DB.Collection("user_data").UpdateOne(ctx, bson.M{"user_id": stytchUser.UserID}, bson.M{"$set": bson.M{"roles": userData.Roles}}); err != nil {
				fmt.Printf("Error updating user data while inviting existing user with ID \"%s\" to a project: %v\n", stytchUser.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// Send user an email saying that they've been invited
			m := sgmail.NewV3Mail()
			m.SetTemplateID(config.I.Email.Templates.InviteExistingUser)
			from := sgmail.NewEmail("Decent", config.I.Email.NoReplyEmail)
			m.SetFrom(from)
			p := sgmail.NewPersonalization()
			to := sgmail.NewEmail(fmt.Sprintf("%s %s", stytchUser.Name.FirstName, stytchUser.Name.LastName), stytchUser.Emails[0].Email)
			p.AddTos(to)
			p.SetDynamicTemplateData("team_name", team.Name)
			m.AddPersonalizations(p)
			sgreq := sendgrid.GetRequest(config.I.Email.SendGridAPIKey, "/v3/mail/send", "https://api.sendgrid.com")
			sgreq.Method = "POST"
			sgreq.Body = sgmail.GetRequestBody(m)
			_, err := sendgrid.API(sgreq)
			if err != nil {
				fmt.Printf("Error sending invite email to existing user with ID \"%s\": %v\n", stytchUser.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		}
	}

	return nil
}

// Check if the specified team name is available to use.
func IsTeamNameAvailable(c *fiber.Ctx) error {
	teamName := c.Params("team_name")

	// Check if team name is available
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var team models.Team
	if err := config.MI.DB.Collection("teams").FindOne(ctx,
		bson.M{"name": teamName},
	).Decode(&team); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"available": true,
			})
		}

		fmt.Printf("Error checking if team name \"%s\" is available: %v\n", teamName, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"available": false,
	})
}
