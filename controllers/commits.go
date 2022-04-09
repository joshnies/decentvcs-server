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

	// Get commits from database
	cur, err := config.MI.DB.Collection("commits").Find(ctx, bson.M{})
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

	// Parse body
	var body models.Commit
	if err := c.BodyParser(&body); err != nil {
		fmt.Println(err) // DEBUG
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

	// Create new commit
	commit := models.Commit{
		Id:        primitive.NewObjectID(),
		CreatedAt: time.Now().Unix(),
		Message:   body.Message,
		BranchId:  body.BranchId,
	}

	// Create commit in database
	_, err := config.MI.DB.Collection("commits").InsertOne(ctx, commit)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(commit)
}

// TODO: Add update route
// TODO: Add delete route
