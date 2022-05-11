package controllers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/config"
	"github.com/joshnies/qc-api/lib/auth"
	"github.com/joshnies/qc-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Get many commits for the given project, regardless of branch.
//
// URL params:
//
// - pid: project ID
//
// Query params:
//
// - before: commit ID to compare with
//
// - after: commit ID to compare with
//
// - limit: number of commits to return
//
func GetManyCommits(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get limit query param
	limitStr := c.Query("limit")
	if limitStr == "" {
		limitStr = "10"
	}
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		limit = 10
	}

	// Get project ID
	projectId, err := primitive.ObjectIDFromHex(c.Params("pid"))
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get compared commit ID as string
	comparedCommitIdStr := c.Query("before")
	if comparedCommitIdStr == "" {
		comparedCommitIdStr = c.Query("after")
	}

	// If "before" or "after" query param set, get it from database
	var comparedCommit models.Commit
	if comparedCommitIdStr != "" {
		comparedCommitId, err := primitive.ObjectIDFromHex(comparedCommitIdStr)
		if err != nil {
			fmt.Println(err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": "Invalid commit ID; must be an ObjectID hexadecimal",
			})
		}

		// Get compared commit from database
		err = config.MI.DB.Collection("commits").FindOne(ctx, bson.M{
			"_id":        comparedCommitId,
			"project_id": projectId,
		}).Decode(&comparedCommit)
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": "No commit found for query param",
			})
		}
	}

	// Build bson filter
	filter := bson.M{"project_id": projectId}

	if comparedCommitIdStr != "" {
		if c.Query("before") != "" {
			// Before
			filter["created_at"] = bson.M{
				"$lt": comparedCommit.CreatedAt,
			}
		} else {
			// After
			filter["created_at"] = bson.M{
				"$gt": comparedCommit.CreatedAt,
			}
		}
	}

	fmt.Printf("Filter: %+v\n", filter)

	// Get commits from mongo
	// Includes branch
	cur, err := config.MI.DB.Collection("commits").Aggregate(ctx, []bson.M{
		{
			"$match": filter,
		},
		{
			"$sort": bson.M{
				"created_at": -1, // ascending
			},
		},
		{
			"$limit": limit,
		},
		{
			"$lookup": bson.M{
				"from":         "branches",
				"localField":   "branch_id",
				"foreignField": "_id",
				"as":           "branch",
			},
		},
		{
			"$unwind": "$branch",
		},
		{
			"$unset": "branch_id",
		},
	})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	defer cur.Close(ctx)

	// Iterate over the results and decode into slice of Commits
	var result []models.CommitWithBranch
	for cur.Next(ctx) {
		var decoded models.CommitWithBranch
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

// Get many commits for the given branch.
//
// URL params:
//
// - pid: project ID
//
// - bid: branch ID
//
// Query params:
//
// - before: commit ID to compare with
//
// - after: commit ID to compare with
//
// - limit: number of commits to return
//
func GetManyCommitsForBranch(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get limit query param
	limitStr := c.Query("limit")
	if limitStr == "" {
		limitStr = "10"
	}
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		limit = 10
	}

	// Get project ID
	projectId, err := primitive.ObjectIDFromHex(c.Params("pid"))
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get branch ID
	branchId, err := primitive.ObjectIDFromHex(c.Params("bid"))
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid branch ID",
		})
	}

	// Get compared commit ID as string
	comparedCommitIdStr := c.Query("before")
	if comparedCommitIdStr == "" {
		comparedCommitIdStr = c.Query("after")
	}

	// If "before" or "after" query param set, get it from database
	var comparedCommit models.Commit
	if comparedCommitIdStr != "" {
		comparedCommitId, err := primitive.ObjectIDFromHex(comparedCommitIdStr)
		if err != nil {
			fmt.Println(err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": "Invalid commit ID; must be an ObjectID hexadecimal",
			})
		}

		// Get compared commit from database
		err = config.MI.DB.Collection("commits").FindOne(ctx, bson.M{
			"_id":        comparedCommitId,
			"project_id": projectId,
			"branch_id":  branchId,
		}).Decode(&comparedCommit)
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": "No commit found for query param",
			})
		}
	}

	// Get commits from database
	filter := bson.M{"project_id": projectId, "branch_id": branchId}

	if comparedCommitIdStr != "" {
		if c.Query("before") != "" {
			// Before
			filter["created_at"] = bson.M{
				"$lt": comparedCommit.CreatedAt,
			}
		} else {
			// After
			filter["created_at"] = bson.M{
				"$gt": comparedCommit.CreatedAt,
			}
		}
	}

	cur, err := config.MI.DB.Collection("commits").Find(ctx, filter,
		options.Find().SetSort(bson.D{{"created_at", 1}}), // ascending
		options.Find().SetLimit(limit),
	)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	defer cur.Close(ctx)

	// Iterate over the results and decode into slice of Commits
	var result []models.Commit
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

// Get one commit by index.
func GetOneCommitByIndex(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get query params
	projectId, err := primitive.ObjectIDFromHex(c.Params("pid"))
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	idx, err := strconv.Atoi(c.Params("idx"))
	if err != nil || idx <= 0 {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid commit index. Must be a positive integer",
		})
	}

	// Get commit from database
	var result models.Commit
	err = config.MI.DB.Collection("commits").FindOne(ctx, bson.M{"project_id": projectId, "index": idx}).Decode(&result)
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

// Get one commit by ID.
func GetOneCommitByID(c *fiber.Ctx) error {
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

	// Get user ID
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

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
	projectId, err := primitive.ObjectIDFromHex(c.Params("pid"))
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID; must be an ObjectID hexadecimal",
		})
	}

	// Create branch ObjectID
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
	var branchWithLastCommit models.BranchWithCommit
	err = cur.Decode(&branchWithLastCommit)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create commit object
	commit := models.Commit{
		ID:            primitive.NewObjectID(),
		CreatedAt:     time.Now().Unix(),
		Index:         branchWithLastCommit.Commit.Index + 1,
		ProjectID:     projectId,
		BranchID:      branchId,
		Message:       reqBody.Message,
		CreatedFiles:  reqBody.CreatedFiles,
		ModifiedFiles: reqBody.ModifiedFiles,
		DeletedFiles:  reqBody.DeletedFiles,
		HashMap:       reqBody.HashMap,
		AuthorID:      userId,
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
