package projects

import (
	"context"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/config"
	"github.com/joshnies/qc-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var validate = validator.New()

// Get many projects.
func GetManyProjects(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var result []models.Project
	defer cancel()

	// Get projects from database
	cur, err := config.MI.DB.Collection("projects").Find(ctx, bson.M{})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Iterate over the results and decode into slice of Projects
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var decodedProject models.Project
		err := cur.Decode(&decodedProject)
		if err != nil {
			fmt.Println(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		result = append(result, decodedProject)
	}

	return c.JSON(result)
}

// Get one project.
func GetOneProject(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result models.Project
	objId, _ := primitive.ObjectIDFromHex(c.Params("id"))

	// Get project from database
	err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"_id": objId}).Decode(&result)
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

// Create a new project.
func CreateProject(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse body
	var body models.Project
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

	// Create new project
	project := models.Project{
		Id:   primitive.NewObjectID(),
		Name: body.Name,
	}

	// Create project in database
	_, err := config.MI.DB.Collection("projects").InsertOne(ctx, project)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(project)
}
