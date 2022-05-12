package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/config"
	"github.com/joshnies/qc-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Get many branches.
func GetManyBranches(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get project ID
	projectId, err := primitive.ObjectIDFromHex(c.Params("pid"))
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Build mongo aggregation pipeline
	pipeline := []bson.M{
		{"$match": bson.M{"project_id": projectId, "deleted_at": bson.M{"$exists": false}}},
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
		fmt.Println(err)
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
			fmt.Println(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		result = append(result, decoded)
	}

	return c.JSON(result)
}

// Get one branch with commit it currently points to.
//
// URL params:
//
// - bid: Branch ID or name
//
// Query params:
//
// - join_commit: Whether to join commit to branch.
func GetOneBranch(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get URL params
	projectId, err := primitive.ObjectIDFromHex(c.Params("pid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	bid := c.Params("bid")
	var filterName string
	objId, err := primitive.ObjectIDFromHex(bid)
	if err != nil {
		filterName = bid
	}

	// Build mongo aggregation pipeline
	pipeline := []bson.M{}

	if objId != primitive.NilObjectID {
		// Filter by ID
		pipeline = append(pipeline, bson.M{"$match": bson.M{"project_id": projectId, "deleted_at": bson.M{"$exists": false}, "_id": objId}})
	} else {
		// Filter by name
		pipeline = append(pipeline, bson.M{"$match": bson.M{"project_id": projectId, "deleted_at": bson.M{"$exists": false}, "name": filterName}})
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
		fmt.Println("Error getting branch:", err)
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
			fmt.Println("Error decoding branch:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		return c.JSON(res)
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Not found",
	})
}

// Create a new branch.
//
// URL params:
//
// - pid: Project ID
//
// Request body:
//
// - name: Branch name
// - commit_index: Index of the commit this branch points to
//
func CreateBranch(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get URL params
	projectId, err := primitive.ObjectIDFromHex(c.Params("pid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Parse body
	var body models.BranchCreateDTO
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Validate body
	if vErr := validate.Struct(body); vErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": vErr.Error(),
		})
	}

	// Get commit by index
	var commit models.Commit
	err = config.MI.DB.Collection("commits").FindOne(ctx, bson.M{"project_id": projectId, "index": body.CommitIndex}).Decode(&commit)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Commit not found",
		})
	}

	// Check if branch already exists
	var branch models.Branch
	err = config.MI.DB.Collection("branches").FindOne(ctx, bson.M{"project_id": projectId, "name": body.Name}).Decode(&branch)
	if err != nil && err != mongo.ErrNoDocuments {
		fmt.Println("Error getting branch:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	if err == nil {
		if branch.DeletedAt == 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Branch already exists",
			})
		} else {
			// Unset "deleted_at" for existing branch
			_, err = config.MI.DB.Collection("branches").UpdateOne(ctx, bson.M{"project_id": projectId, "_id": branch.ID}, bson.M{"$unset": bson.M{"deleted_at": ""}})
			if err != nil {
				fmt.Printf("Error unsetting \"deleted_at\" for existing branch w/ ID \"%s\": %+v\n", branch.ID, err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		}
	} else if errors.Is(err, mongo.ErrNoDocuments) {
		// Create new branch
		branch = models.Branch{
			ID:        primitive.NewObjectID(),
			CreatedAt: time.Now().Unix(),
			Name:      body.Name,
			ProjectID: projectId,
			CommitID:  commit.ID,
		}

		// Create branch in database
		_, err = config.MI.DB.Collection("branches").InsertOne(ctx, branch)
		if err != nil {
			fmt.Println("Error creating new branch:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	return c.JSON(branch)
}

// Soft-delete one branch.
func DeleteOneBranch(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get URL params
	projectId, err := primitive.ObjectIDFromHex(c.Params("pid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get all branches for project
	var branches []models.Branch
	cur, err := config.MI.DB.Collection("branches").Find(ctx, bson.M{"project_id": projectId})
	if err != nil {
		fmt.Println("Error getting branches:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Iterate over the results and decode into slice of Branches
	for cur.Next(ctx) {
		var res models.Branch
		err = cur.Decode(&res)
		if err != nil {
			fmt.Println("Error decoding branch:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	// If there's only one branch, return error
	if len(branches) == 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Cannot delete the only branch in a project",
		})
	}

	// Build mongo pipeline
	bid := c.Params("bid")
	var filterName string
	objId, err := primitive.ObjectIDFromHex(bid)
	if err != nil {
		filterName = bid
	}

	filter := bson.M{"project_id": projectId}

	if objId != primitive.NilObjectID {
		// Filter by ID
		filter["_id"] = objId
	} else {
		// Filter by name
		filter["name"] = filterName
	}

	// Soft-delete branch
	_, err = config.MI.DB.Collection("branches").UpdateOne(ctx, filter, bson.M{"$set": bson.M{"deleted_at": time.Now().Unix()}})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Branch deleted",
	})
}

// TODO: Add update route
