package controllers

import (
	"context"
	"errors"
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

// Get many branches.
func GetManyBranches(c *fiber.Ctx) error {
	// TODO: Add pagination

	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Get project from database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var project models.Project
	if err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": team.ID, "name": projectName}).Decode(&project); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Project not found",
			})
		}

		fmt.Printf("[GetManyBranches] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Build mongo aggregation pipeline
	pipeline := []bson.M{
		{"$match": bson.M{"project_id": project.ID, "deleted_at": bson.M{"$exists": false}}},
	}

	if c.Query("join_commit") == "true" {
		// Join commit
		pipeline = append(pipeline, []bson.M{
			{
				"$lookup": bson.M{
					"from":         "commits",
					"localField":   "commit_id",
					"foreignField": "_id",
					"as":           "commit",
				},
			},
			{
				"$unwind": "$commit",
			},
			{
				"$unset": "commit_id",
			},
		}...)
	}

	// Get branches from database
	cur, err := config.MI.DB.Collection("branches").Aggregate(ctx, pipeline)
	if err != nil {
		fmt.Printf("[GetManyBranches] Error getting branches: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	defer cur.Close(ctx)

	// Iterate over the results and decode into slice of Branches
	var result []models.BranchWithCommit
	for cur.Next(ctx) {
		var decoded models.BranchWithCommit
		err := cur.Decode(&decoded)
		if err != nil {
			fmt.Printf("[GetManyBranches] Error decoding branches: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		result = append(result, decoded)
	}

	return c.JSON(result)
}

// Get one branch.
func GetOneBranch(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")
	branchName := c.Params("branch_name")

	// Get project from database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var project models.Project
	if err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": team.ID, "name": projectName}).Decode(&project); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Project not found",
			})
		}

		fmt.Printf("[GetOneBranch] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Build mongo aggregation pipeline
	pipeline := []bson.M{
		bson.M{
			"$match": bson.M{
				"deleted_at": bson.M{"$exists": false},
				"project_id": project.ID,
				"name":       branchName,
			},
		},
	}

	if c.Query("join_commit") == "true" {
		// Join commit
		pipeline = append(pipeline, []bson.M{
			{
				"$lookup": bson.M{
					"from":         "commits",
					"localField":   "commit_id",
					"foreignField": "_id",
					"as":           "commit",
				},
			},
			{
				"$unwind": "$commit",
			},
			{
				"$unset": "commit_id",
			},
		}...)
	}

	// Get branch from database, including commit it currently points to
	cur, err := config.MI.DB.Collection("branches").Aggregate(ctx, pipeline)
	if err != nil {
		fmt.Printf("[GetOneBranch] Error getting branch: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	defer cur.Close(ctx)

	// Iterate over the results and decode into slice of Branches
	cur.Next(ctx)
	var res models.BranchWithCommit
	err = cur.Decode(&res)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Branch not found",
		})
	}

	return c.JSON(res)
}

// Get the default branch of a project.
func GetDefaultBranch(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var project models.Project
	if err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": team.ID, "name": projectName}).Decode(&project); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Project not found",
			})
		}

		fmt.Printf("[GetDefaultBranch] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Build mongo aggregation pipeline
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"_id":        project.DefaultBranchID,
				"deleted_at": bson.M{"$exists": false},
				"project_id": project.ID,
			},
		},
	}

	if c.Query("join_commit") == "true" {
		// Join commit
		pipeline = append(pipeline, []bson.M{
			{
				"$lookup": bson.M{
					"from":         "commits",
					"localField":   "commit_id",
					"foreignField": "_id",
					"as":           "commit",
				},
			},
			{
				"$unwind": "$commit",
			},
			{
				"$unset": "commit_id",
			},
		}...)
	}

	// Get branch from database, including commit it currently points to
	cur, err := config.MI.DB.Collection("branches").Aggregate(ctx, pipeline)
	if err != nil {
		fmt.Printf("[GetDefaultBranch] Error getting branch: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	defer cur.Close(ctx)

	// Iterate over the results and decode into slice of Branches
	if cur.Next(ctx) {
		var res models.BranchWithCommit
		err = cur.Decode(&res)
		if err != nil {
			fmt.Printf("[GetDefaultBranch] Error decoding branch: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		return c.JSON(res)
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Default branch not found",
	})
}

// Create a new branch.
func CreateBranch(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Parse body
	var body models.BranchCreateDTO
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

	// Validate branch name
	regex := regexp.MustCompile(`^[\w\-]+$`)
	if !regex.MatchString(body.Name) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid branch name; must be alphanumeric with dashes",
		})
	}

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var project models.Project
	if err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": team.ID, "name": projectName}).Decode(&project); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Project not found",
			})
		}

		fmt.Printf("[CreateBranch] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get commit by index
	var commit models.Commit
	err := config.MI.DB.Collection("commits").FindOne(ctx, bson.M{"project_id": project.ID, "index": body.CommitIndex}).Decode(&commit)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Commit not found",
		})
	}

	// Check if branch already exists
	var branch models.Branch
	err = config.MI.DB.Collection("branches").FindOne(ctx, bson.M{"project_id": project.ID, "name": body.Name}).Decode(&branch)
	if err != nil && err != mongo.ErrNoDocuments {
		fmt.Printf("[CreateBranch] Error getting branch: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	if err == nil {
		if branch.DeletedAt.IsZero() {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Branch already exists",
			})
		} else {
			// Unset "deleted_at" for existing branch
			_, err = config.MI.DB.Collection("branches").UpdateOne(ctx, bson.M{"project_id": project.ID, "_id": branch.ID}, bson.M{"$unset": bson.M{"deleted_at": ""}})
			if err != nil {
				fmt.Printf("Error unsetting \"deleted_at\" for existing branch w/ ID \"%s\": %+v\n", branch.ID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		}
	} else if errors.Is(err, mongo.ErrNoDocuments) {
		// Create new branch
		branch := models.BranchCreateBSON{
			ID:        primitive.NewObjectID(),
			CreatedAt: time.Now(),
			Name:      body.Name,
			ProjectID: project.ID,
			CommitID:  commit.ID,
		}

		// Create branch in database
		_, err = config.MI.DB.Collection("branches").InsertOne(ctx, branch)
		if err != nil {
			fmt.Printf("[CreateBranch] Error creating new branch: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		return c.JSON(branch)
	}

	return c.JSON(branch)
}

// Update a branch.
func UpdateBranch(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")
	branchName := c.Params("branch_name")

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var project models.Project
	if err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": team.ID, "name": projectName}).Decode(&project); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Project not found",
			})
		}

		fmt.Printf("[UpdateOneBranch] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Parse body
	var body models.BranchUpdateDTO
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Update branch
	filter := bson.M{"project_id": project.ID, "name": branchName}
	_, err := config.MI.DB.Collection("branches").UpdateOne(
		ctx,
		filter,
		bson.M{"$set": bson.M{"name": body.Name}},
	)
	if err != nil {
		fmt.Printf("[UpdateBranch] Error updating branch: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Branch updated successfully",
	})
}

// Soft-delete one branch.
func SoftDeleteOneBranch(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")
	branchName := c.Params("branch_name")

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var project models.Project
	if err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": team.ID, "name": projectName}).Decode(&project); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Project not found",
			})
		}

		fmt.Printf("[SoftDeleteOneBranch] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get all branches for project
	var branches []models.Branch
	cur, err := config.MI.DB.Collection("branches").Find(ctx, bson.M{"project_id": project.ID})
	if err != nil {
		fmt.Printf("[SoftDeleteOneBranch] Error getting branches: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Iterate over the results and decode into slice of Branches
	for cur.Next(ctx) {
		var res models.Branch
		err = cur.Decode(&res)
		if err != nil {
			fmt.Printf("[SoftDeleteOneBranch] Error decoding branch: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	// If there's only one branch, return error
	if len(branches) == 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete the only branch in a project",
		})
	}

	// Soft-delete branch
	filter := bson.M{"project_id": project.ID, "name": branchName}
	_, err = config.MI.DB.Collection("branches").UpdateOne(ctx, filter, bson.M{"$set": bson.M{"deleted_at": time.Now()}})
	if err != nil {
		fmt.Printf("[SoftDeleteOneBranch] Error soft-deleting branch: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Branch deleted successfully",
	})
}
