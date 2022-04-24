package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/config"
	"github.com/joshnies/qc-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"storj.io/uplink"
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

	var result models.Commit
	objId, _ := primitive.ObjectIDFromHex(c.Params("id"))

	// Get commit from database
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
func CreateCommit(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get file URIs
	fileURIs := strings.Split(c.FormValue("file_uris"), ",")

	if len(fileURIs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "At least one file URI is required",
		})
	}

	// TODO: Validate file URIs

	// Create project ObjectID
	projectId, err := primitive.ObjectIDFromHex(c.Params("pid"))
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid branch_id; must be an ObjectID hexadecimal",
		})
	}

	// Create commit object
	commit := models.Commit{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().Unix(),
		Message:   c.FormValue("message"),
		ProjectID: projectId,
		FileURIs:  fileURIs,
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

	return c.JSON(commit)
}

// Get presigned URLs to upload objects to storage from a client.
func GetAccessGrant(c *fiber.Ctx) error {
	// Get params
	permission := uplink.ReadOnlyPermission()
	if c.Params("permission") == "w" {
		permission = uplink.WriteOnlyPermission()
	}

	// TODO: Return unauthorized if user is not logged in

	// Get commit
	commitId, _ := primitive.ObjectIDFromHex(c.Params("id"))
	commit := models.Commit{}
	err := config.MI.DB.Collection("commits").FindOne(context.Background(), bson.M{"_id": commitId}).Decode(&commit)
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

	// Get project
	project := models.Project{}
	err = config.MI.DB.Collection("projects").FindOne(context.Background(), bson.M{"_id": commit.ProjectID}).Decode(&project)
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

	// TODO: Return unauthorized if user is not authorized to access project

	// Create uplink access grant
	access, err := config.SI.Access.Share(permission, uplink.SharePrefix{
		Bucket: config.SI.Bucket,
		Prefix: fmt.Sprintf("%s/%s", commit.ProjectID, commit.ID),
	})

	if err != nil {
		println(fmt.Sprintf("Failed to create access grant to Storj bucket %s: %s", config.SI.Bucket, err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Serialize restricted access grant so it can be used later with `ParseAccess()` (or equiv.)
	// by the client
	accessSerialized, err := access.Serialize()
	if err != nil {
		println(fmt.Sprintf("Failed to serialize access grant to Storj bucket %s: %s", config.SI.Bucket, err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"access": accessSerialized,
	})
}

// TODO: Add update route
// TODO: Add delete route
