package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/config"
	"github.com/joshnies/qc-api/models"
	"github.com/lucsky/cuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Get many commits.
func GetManyCommits(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var result []models.Commit
	defer cancel()

	// NOTE: Commented out since it's currently unused
	// Get project ID
	// projectId, err := primitive.ObjectIDFromHex(c.Params("pid"))
	// if err != nil {
	// 	fmt.Println(err)
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"error":   "Bad request",
	// 		"message": "Invalid project ID",
	// 	})
	// }

	// Get branch ID
	branchId, err := primitive.ObjectIDFromHex(c.Params("bid"))
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid branch ID",
		})
	}

	// Get commits from database
	cur, err := config.MI.DB.Collection("commits").Find(ctx, bson.M{"branch_id": branchId})
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
	objId, _ := primitive.ObjectIDFromHex(c.Params("cid"))

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

	// Generate file key
	fileKey := cuid.New()

	// Create branch ObjectID
	branch_id, err := primitive.ObjectIDFromHex(c.FormValue("branch_id"))
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid branch_id; must be an ObjectID hexadecimal",
		})
	}

	// Create commit object
	commit := models.Commit{
		Id:        primitive.NewObjectID(),
		CreatedAt: time.Now().Unix(),
		Message:   c.FormValue("message"),
		BranchId:  branch_id,
		FileKey:   fileKey,
	}

	// Get file
	file, err := c.FormFile("file")
	if err != nil {
		fmt.Println(err) // DEBUG
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Missing or invalid file",
		})
	}

	// Open file
	iofile, err := file.Open()
	if err != nil {
		fmt.Println("Failed to read file while creating commit. Aborting.")
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Upload file to Linode Object Storage
	_, err = config.SI.Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: &config.SI.Bucket,
		Key:    &fileKey,
		Body:   iofile,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == request.CanceledErrorCode {
			fmt.Println("Commit upload canceled due to timeout.")
			fmt.Println(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		fmt.Println("Commit upload failed.")
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Insert commit into database
	_, err = config.MI.DB.Collection("commits").InsertOne(ctx, commit)
	if err != nil {
		// Attempt to delete file from storage
		_, err = config.SI.Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
			Bucket: &config.SI.Bucket,
			Key:    &fileKey,
		})
		if err != nil {
			fmt.Println("Failed to delete uploaded commit file from storage after a failed commit insert into database.")
			fmt.Println(err)
		}

		// Log and return error
		fmt.Println("Failed to insert commit into database. File was still uploaded.")
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(commit)
}

// TODO: Add update route
// TODO: Add delete route