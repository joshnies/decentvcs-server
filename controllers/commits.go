package controllers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/auth"
	"github.com/joshnies/decent-vcs/lib/branch_lib"
	"github.com/joshnies/decent-vcs/lib/team_lib"
	"github.com/joshnies/decent-vcs/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Get many commits for the given project.
func GetManyCommits(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Get limit query param
	limitStr := c.Query("limit")
	if limitStr == "" {
		limitStr = "10"
	}
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		limit = 10
	}

	// Get compared commit ID as string
	comparedCommitIdStr := c.Query("before")
	if comparedCommitIdStr == "" {
		comparedCommitIdStr = c.Query("after")
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

		fmt.Printf("[GetManyCommits] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// If "branch_name" query param set, get branch from database
	var branch models.Branch
	branchName := c.Query("branch_name")
	if branchName != "" {
		if err := config.MI.DB.Collection("branches").FindOne(ctx, bson.M{"project_id": project.ID, "name": branchName}).Decode(&branch); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Branch not found",
				})
			}

			fmt.Printf("[GetManyCommits] Error getting branch: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	// If "before" or "after" query param set, get it from database
	var comparedCommit models.Commit
	if comparedCommitIdStr != "" {
		comparedCommitId, err := primitive.ObjectIDFromHex(comparedCommitIdStr)
		if err != nil {
			fmt.Printf("[GetManyCommits] Error getting compared commit: %v\n", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": "Invalid commit ID; must be an ObjectID hexadecimal",
			})
		}

		// Get compared commit from database
		err = config.MI.DB.Collection("commits").FindOne(ctx, bson.M{
			"_id":        comparedCommitId,
			"project_id": project.ID,
		}).Decode(&comparedCommit)
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": "No commit found for query param",
			})
		}
	}

	// Build bson filter
	filter := bson.M{"project_id": project.ID}

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

	if branchName != "" {
		filter["branch_id"] = branch.ID
	}

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
		fmt.Printf("[GetManyCommits] Error getting commits: %v\n", err)
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
			fmt.Printf("[GetManyCommits] Error decoding commits: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		result = append(result, decoded)
	}

	return c.JSON(result)
}

// Get one commit by index.
func GetOneCommit(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Get branch index
	idx, err := strconv.Atoi(c.Params("commit_index"))
	if err != nil || idx <= 0 {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid commit index. Must be a positive non-zero integer",
		})
	}

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

		fmt.Printf("[GetOneCommit] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get commit from database
	var result models.Commit
	err = config.MI.DB.Collection("commits").FindOne(ctx, bson.M{"project_id": project.ID, "index": idx}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Not found",
		})
	}
	if err != nil {
		fmt.Printf("[GetOneCommit] Error getting commit: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(result)
}

// Create a new commit and update team usage metrics for billing purposes.
func CreateCommit(c *fiber.Ctx) error {
	userData := auth.GetUserDataFromContext(c)
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")
	branchName := c.Params("branch_name")

	// Parse request body
	var reqBody models.CreateCommitRequest
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

		fmt.Printf("[CreateCommit] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get branch with commit
	branch, err := branch_lib.GetOneWithCommit(team.ID, projectName, branchName)
	if err != nil {
		fmt.Printf("[CreateCommit] Error getting branch: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Check if any created/modified/delete file in new commit references a file locked by another user
	combinedFiles := reqBody.CreatedFiles
	combinedFiles = append(combinedFiles, reqBody.ModifiedFiles...)
	combinedFiles = append(combinedFiles, reqBody.DeletedFiles...)
	for _, path := range combinedFiles {
		if lockedBy, ok := branch.Locks[path]; ok {
			if lockedBy != userData.UserID {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("File \"%s\" is locked by %s", path, lockedBy),
				})
			}
		}
	}

	// Create commit object
	commit := models.Commit{
		ID:            primitive.NewObjectID(),
		CreatedAt:     time.Now(),
		Index:         branch.Commit.Index + 1,
		ProjectID:     project.ID,
		BranchID:      branch.ID,
		Message:       reqBody.Message,
		CreatedFiles:  reqBody.CreatedFiles,
		ModifiedFiles: reqBody.ModifiedFiles,
		DeletedFiles:  reqBody.DeletedFiles,
		HashMap:       reqBody.HashMap,
		AuthorID:      userData.UserID,
	}

	// Insert commit into database
	if _, err = config.MI.DB.Collection("commits").InsertOne(ctx, commit); err != nil {
		fmt.Printf("[CreateCommit] Failed to insert commit into database: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Update branch to point to new commit
	if _, err = config.MI.DB.Collection("branches").UpdateOne(ctx, bson.M{"_id": branch.ID}, bson.M{"$set": bson.M{"commit_id": commit.ID}}); err != nil {
		fmt.Printf("[CreateCommit] Failed to update branch: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get file sizes from storage
	var storedFiles []string
	storedFiles = append(storedFiles, reqBody.CreatedFiles...)
	storedFiles = append(storedFiles, reqBody.ModifiedFiles...)

	for _, path := range storedFiles {
		hash := commit.HashMap[path]

		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		storageKey := FormatStorageKey(*team, project, hash)
		s3Res, err := config.SI.Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: &config.SI.ProjectsBucket,
			Key:    &storageKey,
		})
		if err != nil {
			fmt.Printf("[CreateCommit] Error getting file size of object \"%s\": %v\n", storageKey, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		// Calculate and add file size, rounded up to the nearest MB
		team.StorageUsedMB += float64(s3Res.ContentLength) / 1024 / 1024
	}

	// Update team storage usage in database
	if _, err = config.MI.DB.Collection("teams").UpdateOne(ctx, bson.M{"_id": team.ID}, bson.M{"$set": bson.M{"storage_used_mb": team.StorageUsedMB}}); err != nil {
		fmt.Printf("[CreateCommit] Failed to update team: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(commit)
}

// Update a commit.
func UpdateCommit(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")
	commitIndex := c.Params("commit_index")

	// Parse request body
	var commit models.Commit
	if err := c.BodyParser(&commit); err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

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

		fmt.Printf("[UpdateOneCommit] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Update commit in database
	if _, err := config.MI.DB.Collection("commits").UpdateOne(ctx, bson.M{"project_id": project.ID, "index": commitIndex}, bson.M{"$set": commit}); err != nil {
		fmt.Printf("[UpdateCommit] Error updating commit: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Updated commit successfully",
	})
}

// Delete many commits after the specified index in the specified branch.
func DeleteManyCommitsInBranch(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")
	branchName := c.Params("branch_name")

	// Get "after" query param
	after, err := strconv.Atoi(c.Query("after"))
	if err != nil || after <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query param \"after\"; must be a positive non-zero integer",
		})
	}

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

		fmt.Printf("[UpdateOneCommit] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get branch from database
	var branch models.Branch
	if err := config.MI.DB.Collection("branches").FindOne(ctx, bson.M{"project_id": project.ID, "name": branchName}).Decode(&branch); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Branch not found",
			})
		}

		fmt.Printf("[UpdateOneCommit] Error getting branch: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get commit with index
	var afterCommit models.Commit
	if err = config.MI.DB.Collection("commits").FindOne(ctx, bson.M{"branch_id": branch.ID, "index": after}).Decode(&afterCommit); err != nil {
		fmt.Printf("[DeleteManyCommitsInBranch] Error getting commit with index: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Update branch to point to commit with specified index
	if _, err := config.MI.DB.Collection("branches").UpdateOne(ctx, bson.M{"_id": branch.ID}, bson.M{"$set": bson.M{"commit_id": afterCommit.ID}}); err != nil {
		fmt.Printf("[DeleteManyCommitsInBranch] Error updating branch: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete commits after specified index
	if _, err := config.MI.DB.Collection("commits").DeleteMany(ctx, bson.M{"branch_id": branch.ID, "index": bson.M{"$gt": after}}); err != nil {
		fmt.Printf("[DeleteManyCommitsInBranch] Error deleting many commits after: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Deleted commits successfully",
	})
}
