package controllers

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/team_lib"
	"github.com/joshnies/decent-vcs/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Get one project by ID.
func GetOneProject(c *fiber.Ctx) error {
	// Get team from context
	team := team_lib.GetTeamFromContext(c)

	// Get URL params
	projectName := c.Params("project_name")

	// Get project from database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result models.Project
	err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": team.ID, "name": projectName}).Decode(&result)
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

// Create a new project. Only team admins can create new projects for the team.
func CreateProject(c *fiber.Ctx) error {
	fmt.Printf("[DEBUG] CreateProject route\n")

	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := config.MI.DB.Collection("projects").InsertOne(ctx, project); err != nil {
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

	if _, err := config.MI.DB.Collection("commits").InsertOne(ctx, commit); err != nil {
		// Delete project
		config.MI.DB.Collection("projects").DeleteOne(ctx, bson.M{"_id": project.ID})

		// Output error
		fmt.Printf("[CreateProject] Error creating initial commit: %v\n", err)
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
	if _, err := config.MI.DB.Collection("branches").InsertOne(ctx, branch); err != nil {
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
func UpdateProject(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Parse request body
	var body models.UpdateProjectRequest
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
				"error": "Invalid name; must be alphanumeric with dashes",
			})
		}

		updateData["name"] = body.Name
	}
	if body.DefaultBranchID != "" {
		defBranchID, err := primitive.ObjectIDFromHex(body.DefaultBranchID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid default branch ID; must be a valid hexadecimal string",
			})
		}

		updateData["default_branch_id"] = defBranchID
	}

	// Update project
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := config.MI.DB.Collection("projects").UpdateOne(
		ctx,
		bson.M{"team_id": team.ID, "name": projectName},
		bson.M{"$set": updateData},
	)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(updateData)
}

// Delete project and all of its subresources.
func DeleteOneProject(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Get project
	project := models.Project{}
	err := config.MI.DB.Collection("projects").FindOne(context.Background(), bson.M{"team_id": team.ID, "name": projectName}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Project not found",
		})
	}
	if err != nil {
		fmt.Printf("[DeleteOneProject] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// ---- From here on out, it's been validated that the user has access to the project

	// Delete all commits for project
	_, err = config.MI.DB.Collection("commits").DeleteMany(context.Background(), bson.M{"project_id": project.ID})
	if err != nil {
		fmt.Printf("[DeleteOneProject] Error deleting all commits for project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete all branches for project
	_, err = config.MI.DB.Collection("branches").DeleteMany(context.Background(), bson.M{"project_id": project.ID})
	if err != nil {
		fmt.Printf("[DeleteOneProject] Error deleting all branches for project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete project
	_, err = config.MI.DB.Collection("projects").DeleteOne(context.Background(), bson.M{"_id": project.ID})
	if err != nil {
		fmt.Printf("[DeleteOneProject] Error deleting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Project deleted successfully",
	})
}
