package projects

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/db"
	"github.com/joshnies/qc-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Get many projects.
func GetManyProjects(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var result []models.Project
	defer cancel()

	// Get projects from database
	cur, err := db.DB.Collection("projects").Find(ctx, bson.M{})
	if err != nil {
		fmt.Println(err)
		return c.Status(http.StatusInternalServerError).SendString("Internal server error")
	}

	// Iterate over the results and decode into slice of Projects
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var decodedProject models.Project
		err := cur.Decode(&decodedProject)
		if err != nil {
			fmt.Println(err)
			return c.Status(http.StatusInternalServerError).SendString("Internal server error")
		}

		result = append(result, decodedProject)
	}

	return c.JSON(result)
}

// Get one project.
func GetOneProject(c *fiber.Ctx) error {
	ctx := context.Background()
	var result models.Project

	// Get project from database
	objId, _ := primitive.ObjectIDFromHex(c.Params("id"))
	err := db.DB.Collection("projects").FindOne(ctx, bson.M{"_id": objId}).Decode(&result)
	if err != nil {
		fmt.Println(err)
		return c.Status(http.StatusInternalServerError).SendString("Internal server error")
	}

	return c.JSON(result)
}
