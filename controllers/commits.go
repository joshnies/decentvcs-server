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

// Get many commits.
func GetManyCommits(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var result []models.Commit
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

	// Get commits from database
	cur, err := config.MI.DB.Collection("commits").Find(ctx, bson.M{"project_id": projectId})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Iterate over the results and decode into slice of Commits
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var decoded models.Commit
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

// Get one commit.
func GetOneCommit(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get query params
	objId, _ := primitive.ObjectIDFromHex(c.Params("cid"))

	// Get commit from database
	var result models.Commit
	err := config.MI.DB.Collection("commits").FindOne(ctx, bson.M{"_id": objId}).Decode(&result)
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

// Create a new commit.
func CreateOneCommit(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse request body
	var reqBody models.CommitSerialized
	if err := c.BodyParser(&reqBody); err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid request body",
		})
	}

	// Make sure a branch ID was specified
	if reqBody.BranchID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "branch_id is required",
		})
	}

	// TODO: Validate file paths

	// Create project ObjectID
	branchId, err := primitive.ObjectIDFromHex(reqBody.BranchID)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid branch_id; must be an ObjectID hexadecimal",
		})
	}

	// Get branch with commit
	cur, err := config.MI.DB.Collection("branches").Aggregate(ctx, []bson.M{
		{
			"$match": bson.M{
				"_id": branchId,
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
	var branchWithPreviousCommit models.BranchWithCommit
	err = cur.Decode(&branchWithPreviousCommit)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create commit object
	commit := models.Commit{
		ID:               primitive.NewObjectID(),
		CreatedAt:        time.Now().Unix(),
		PreviousCommitID: branchWithPreviousCommit.Commit.ID,
		BranchID:         branchId,
		Message:          reqBody.Message,
		SnapshotPaths:    reqBody.SnapshotPaths,
		PatchPaths:       reqBody.PatchPaths,
		DeletedPaths:     reqBody.DeletedPaths,
		HashMap:          reqBody.HashMap,
	}

	// Insert commit into database
	_, err = config.MI.DB.Collection("commits").InsertOne(ctx, commit)
	if err != nil {
		fmt.Println("Failed to insert commit into database.")
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Update branch to point to new commit
	_, err = config.MI.DB.Collection("branches").UpdateOne(ctx, bson.M{"_id": branchId}, bson.M{"$set": bson.M{"commit_id": commit.ID}})
	if err != nil {
		fmt.Println("Failed to update branch.")
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(commit)
}

// Update one commit.
func UpdateOneCommit(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse request body
	var commit models.Commit
	if err := c.BodyParser(&commit); err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid request body",
		})
	}

	// Create commit ObjectID
	commitId, err := primitive.ObjectIDFromHex(c.Params("cid"))
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid commit ID; must be an ObjectID hexadecimal",
		})
	}

	// Update commit in database
	_, err = config.MI.DB.Collection("commits").UpdateOne(ctx, bson.M{"_id": commitId}, bson.M{"$set": commit})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	commit.ID = commitId
	return c.JSON(commit)
}

// TODO: Add delete route
