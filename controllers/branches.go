package controllers

import (
	"context"
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
	var result []models.Branch
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

	// Get branches from database
	cur, err := config.MI.DB.Collection("branches").Find(ctx, bson.M{"project_id": projectId})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Iterate over the results and decode into slice of Branches
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var decoded models.Branch
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

// Get one branch.
//
// URL params:
//
// - bid: Branch ID
//
func GetOneBranch(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get query params
	objId, _ := primitive.ObjectIDFromHex(c.Params("bid"))

	// Get branch from database
	var result models.Branch
	err := config.MI.DB.Collection("branches").FindOne(ctx, bson.M{"_id": objId}).Decode(&result)
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

// Get one branch with commit it currently points to.
//
// URL params:
//
// - bid: Branch ID
//
func GetOneBranchWithCommit(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get query params
	objId, _ := primitive.ObjectIDFromHex(c.Params("bid"))

	// Get branch from database, including commit it currently points to
	cur, err := config.MI.DB.Collection("branches").Aggregate(ctx, []bson.M{
		{
			"$match": bson.M{
				"_id": objId,
			},
		},
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
	})
	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(res)
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
	err := config.MI.DB.Collection("commits").FindOne(ctx, bson.M{"index": body.CommitIndex}).Decode(&commit)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Commit not found",
		})
	}

	// Create new branch
	branch := models.Branch{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().Unix(),
		Name:      body.Name,
		CommitID:  commit.ID,
	}

	// Create branch in database
	_, err = config.MI.DB.Collection("branches").InsertOne(ctx, branch)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(branch)
}

// TODO: Add update route
// TODO: Add delete route
