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

	fmt.Println("GetManyBranches") // DEBUG

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
func GetOneBranch(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result models.Branch
	objId, _ := primitive.ObjectIDFromHex(c.Params("bid"))

	// Get branch from database
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

// Create a new branch.
func CreateBranch(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse body
	var body models.Branch
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

	// Create new branch
	branch := models.Branch{
		Id:        primitive.NewObjectID(),
		CreatedAt: time.Now().Unix(),
		Name:      body.Name,
		ProjectId: body.ProjectId,
	}

	// Create branch in database
	_, err := config.MI.DB.Collection("branches").InsertOne(ctx, branch)
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
