package controllers

import (
	"context"
	"fmt"
	"net/mail"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/acl"
	"github.com/joshnies/decent-vcs/lib/auth"
	"github.com/joshnies/decent-vcs/models"
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
	pid := c.Params("pid")
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
//
// Body:
//
// - name: Project name
//
func CreateProject(c *fiber.Ctx) error {
	// Get user ID
	userID, err := auth.GetUserID(c)
	if err != nil {
		return err
	}

	// Parse body
	var body models.Project
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Validate body
	if err := validate.Struct(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
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
	teamID := body.TeamID
	if teamID.IsZero() {
		teamID = userData.DefaultTeamID
	}

	var team models.Team
	if err := config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"_id": teamID}).Decode(&team); err != nil {
		fmt.Printf("Error getting default team for user with ID \"%s\": %v\n", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Generate default branch ID ahead of time
	branchId := primitive.NewObjectID()

	// Create new project
	project := models.Project{
		ID:              primitive.NewObjectID(),
		CreatedAt:       time.Now().Unix(),
		Name:            body.Name,
		Blob:            fmt.Sprintf("%s/%s", team.Name, body.Name),
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
		CreatedAt: time.Now().Unix(),
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
		CreatedAt: time.Now().Unix(),
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
		"_id":  project.ID.Hex(),
		"name": project.Name,
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
	pid := c.Params("pid")
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
		updateData["name"] = body.Name
	}
	if body.DefaultBranchID != primitive.NilObjectID {
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
	pid := c.Params("pid")
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
	pid := c.Params("pid")
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

	// TODO: Invite new users and add permission for existing users

	return nil
}
