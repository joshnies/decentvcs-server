package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/acl"
	"github.com/joshnies/decent-vcs/lib/auth"
	"github.com/joshnies/decent-vcs/lib/team_lib"
	"github.com/joshnies/decent-vcs/models"
	"github.com/sendgrid/sendgrid-go"
	sgmail "github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/stytchauth/stytch-go/v5/stytch"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Get one project by ID.
//
// URL params:
//
// - id: Project ID
//
func GetOneProject(c *fiber.Ctx) error {
	// Get project ID
	pid := c.Params("project_name")
	projectObjId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get project from database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result models.Project
	err = config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"_id": projectObjId}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Not found",
		})
	}
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(result)
}

// Get one project by blob.
//
// URL params:
//
// - oa: Alias of the user or team who owns the project
//
// - pname: Name of the project
//
func GetOneProjectByBlob(c *fiber.Ctx) error {
	// ownerAlias := c.Params("oa")
	projectName := c.Params("pname")

	// Get project from database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result models.Project
	err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"name": projectName}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Not found",
		})
	}
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Check if user has access to project
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	hasAccess, err := acl.HasProjectAccess(userId, result.ID.Hex(), models.RoleNone)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	if !hasAccess {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	return c.JSON(result)
}

// Create a new project.
// Only team admins can create new projects for the team.
//
// Body: `CreateProjectRequest`
//
func CreateProject(c *fiber.Ctx) error {
	// Get user ID
	userID, err := auth.GetUserID(c)
	if err != nil {
		return err
	}

	// Parse body
	var body models.CreateProjectRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Validate blob
	regex := regexp.MustCompile(`^(?:[\w\-]+\/)?[\w\-]+$`)
	if !regex.MatchString(body.Blob) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid blob; must be in the format of \"<team_name>/<project_name>\" (team name is optional)",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": userID}).Decode(&userData); err != nil {
		fmt.Printf("Error getting user data while creating a new project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get default team for user
	// NOTE: Currently only the owner of a team can create a project in it
	blob := body.Blob
	var projectName string
	var teamName string
	if strings.Contains(blob, "/") {
		parts := strings.Split(blob, "/")
		projectName = parts[0]
		teamName = parts[1]
	} else {
		projectName = blob
	}

	var teamFilter bson.M
	if teamName == "" {
		// Team name not provided, fetch by default team ID
		teamFilter = bson.M{"_id": userData.DefaultTeamID}
	} else {
		// Team name provided, fetch by team name
		teamFilter = bson.M{"name": teamName, "owner_user_id": userID}
	}

	var team models.Team
	if err := config.MI.DB.Collection("teams").FindOne(ctx, teamFilter).Decode(&team); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "Not found",
				"message": "Team not found",
			})
		}

		fmt.Printf("Error getting default team for user with ID \"%s\": %v\n", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Make sure user has access to team
	hasAccess, err := acl.HasTeamAccess(userID, team.ID.Hex(), models.RoleAdmin)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	if !hasAccess {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Forbidden",
		})
	}

	// Generate default branch ID ahead of time
	branchId := primitive.NewObjectID()

	// Create new project
	project := models.Project{
		ID:              primitive.NewObjectID(),
		CreatedAt:       time.Now(),
		Name:            projectName,
		Blob:            fmt.Sprintf("%s/%s", team.Name, projectName),
		TeamID:          team.ID,
		DefaultBranchID: branchId,
	}

	// Create project in database
	if _, err = config.MI.DB.Collection("projects").InsertOne(ctx, project); err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create initial commit
	commit := models.Commit{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now(),
		Index:     1, // Starts at 1 since in Go, 0 is the default and used to check for empty values
		ProjectID: project.ID,
		BranchID:  branchId,
		Message:   "Initial commit",
	}

	if _, err = config.MI.DB.Collection("commits").InsertOne(ctx, commit); err != nil {
		// Delete project
		config.MI.DB.Collection("projects").DeleteOne(ctx, bson.M{"_id": project.ID})

		// Output error
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create default branch
	branch := models.BranchCreateBSON{
		ID:        branchId,
		CreatedAt: time.Now(),
		Name:      "stable",
		ProjectID: project.ID,
		CommitID:  commit.ID,
	}

	// Insert branch into database
	if _, err = config.MI.DB.Collection("branches").InsertOne(ctx, branch); err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Add owner role to user for project
	if _, err = acl.AddRole(userID, project.ID, models.RoleOwner); err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"_id":     project.ID.Hex(),
		"name":    project.Name,
		"blob":    project.Blob,
		"team_id": project.TeamID.Hex(),
		"branches": []fiber.Map{
			{
				"_id":  branch.ID.Hex(),
				"name": branch.Name,
				"commit": fiber.Map{
					"_id":   commit.ID.Hex(),
					"index": commit.Index,
				},
			},
		},
	})
}

// Update a project.
//
// URL params:
//
// - pid: Project ID
//
// Body: (any field from Project)
//
// Returns the updated project.
func UpdateOneProject(c *fiber.Ctx) error {
	// Parse project ID
	pid := c.Params("project_name")
	projectObjectId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Make sure user is a project admin
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	hasAccess, err := acl.HasProjectAccess(userId, pid, models.RoleAdmin)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	if !hasAccess {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse body
	var body models.Project
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	updateData := bson.M{}
	if body.Name != "" {
		// Validate
		regex := regexp.MustCompile(`^[\w\-]+$`)
		if !regex.MatchString(body.Name) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": "Invalid name; must be alphanumeric with dashes",
			})
		}

		updateData["name"] = body.Name
	}
	if !body.DefaultBranchID.IsZero() {
		updateData["default_branch_id"] = body.DefaultBranchID
	}

	// Update project
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = config.MI.DB.Collection("projects").UpdateOne(
		ctx,
		bson.M{"_id": projectObjectId},
		bson.M{"$set": updateData},
	)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"_id":               projectObjectId.Hex(),
		"name":              body.Name,
		"default_branch_id": body.DefaultBranchID.Hex(),
	})
}

// Delete project and all of its subresources.
func DeleteOneProject(c *fiber.Ctx) error {
	// Get project
	pid := c.Params("project_name")
	projectObjectId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get project
	project := models.Project{}
	err = config.MI.DB.Collection("projects").FindOne(context.Background(), bson.M{"_id": projectObjectId}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Project not found",
		})
	}
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete all commits for project
	_, err = config.MI.DB.Collection("commits").DeleteMany(context.Background(), bson.M{"project_id": project.ID})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete all branches for project
	_, err = config.MI.DB.Collection("branches").DeleteMany(context.Background(), bson.M{"project_id": project.ID})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete project
	_, err = config.MI.DB.Collection("projects").DeleteOne(context.Background(), bson.M{"_id": projectObjectId})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Project deleted",
	})
}

// Invite many users to a project.
func InviteManyUsers(c *fiber.Ctx) error {
	// Validate project
	pid := c.Params("project_name")
	projectObjId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Parse request body
	var body models.InviteManyUsersDTO
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
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

	// Get project
	project := models.Project{}
	err = config.MI.DB.Collection("projects").FindOne(context.Background(), bson.M{"_id": projectObjId}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Project not found",
		})
	}
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
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

			// Create default team in database
			team, err := team_lib.CreateDefault(inviteRes.UserID, email)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// Create user data in database with project role
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			userData := models.UserData{
				UserID:        inviteRes.UserID,
				DefaultTeamID: team.ID,
				Roles: []models.RoleObject{
					{
						Role:      models.RoleCollab,
						ProjectID: project.ID,
					},
				},
			}
			if _, err := config.MI.DB.Collection("user_data").InsertOne(ctx, userData); err != nil {
				fmt.Printf("Error creating user data for new invited user with ID \"%s\": %v\n", inviteRes.UserID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
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
				if r.ProjectID.Hex() == project.ID.Hex() {
					skip = true
					if config.I.Debug {
						fmt.Printf("Skipped inviting user with ID \"%s\" to project \"%s\" since they already have a role for the project\n", project.Blob, stytchUser.UserID)
					}

					break
				}
			}
			if skip {
				continue
			}

			// Add project role to user data
			userData.Roles = append(userData.Roles, models.RoleObject{
				Role:      models.RoleCollab,
				ProjectID: project.ID,
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
			p.SetDynamicTemplateData("project_name", project.Name)
			p.SetDynamicTemplateData("project_blob", project.Blob)
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
