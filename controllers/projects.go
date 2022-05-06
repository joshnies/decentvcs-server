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
	"storj.io/uplink"
)

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
//
// Query params:
//
// - id: Project ID
//
func GetOneProject(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result models.Project
	objId, _ := primitive.ObjectIDFromHex(c.Params("pid"))

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
//
// Body:
//
// - name: Project name
//
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
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().Unix(),
		Name:      body.Name,
	}

	// Create project in database
	_, err := config.MI.DB.Collection("projects").InsertOne(ctx, project)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Generate default branch ID ahead of time
	branchId := primitive.NewObjectID()

	// Create initial commit
	commit := models.Commit{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().Unix(),
		ProjectID: project.ID,
		BranchID:  branchId,
		Message:   "Initial commit",
	}

	_, err = config.MI.DB.Collection("commits").InsertOne(ctx, commit)
	if err != nil {
		// Delete project
		config.MI.DB.Collection("projects").DeleteOne(ctx, bson.M{"_id": project.ID})

		// Output error
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create default branch
	branch := models.Branch{
		ID:        branchId,
		CreatedAt: time.Now().Unix(),
		Name:      "production",
		CommitID:  commit.ID,
	}

	// Insert branch into database
	_, err = config.MI.DB.Collection("branches").InsertOne(ctx, branch)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"_id":  project.ID.Hex(),
		"name": project.Name,
		"branches": []fiber.Map{
			{
				"_id":  branch.ID.Hex(),
				"name": branch.Name,
				"commits": []fiber.Map{
					{
						"_id":     commit.ID.Hex(),
						"message": commit.Message,
					},
				},
			},
		},
	})
}

// Get Storj access grant for project.
//
// URL params:
//
// - id: Project ID
//
func GetAccessGrant(c *fiber.Ctx) error {
	// TODO: Return unauthorized if user is not logged in

	// Get project
	// TODO: Add user ID to FindOne filter to prevent users from accessing other projects
	pid := c.Params("pid")
	projectObjectId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pid",
		})
	}

	project := models.Project{}
	err = config.MI.DB.Collection("projects").FindOne(context.Background(), bson.M{"_id": projectObjectId}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Project not found",
		})
	}
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create uplink access grant
	access, err := config.SI.Access.Share(uplink.FullPermission(), uplink.SharePrefix{
		Bucket: config.SI.Bucket,
		Prefix: pid,
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
		"access_grant": accessSerialized,
	})
}

// TODO: Add update route

// Delete project and all of its subresources.
func DeleteProject(c *fiber.Ctx) error {
	// TODO: Return unauthorized if user is not logged in

	// Get project
	// TODO: Add user ID to FindOne filter to prevent users from accessing other projects
	pid := c.Params("pid")
	projectObjectId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pid",
		})
	}

	project := models.Project{}
	err = config.MI.DB.Collection("projects").FindOne(context.Background(), bson.M{"_id": projectObjectId}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Project not found",
		})
	}
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete all commits for project
	_, err = config.MI.DB.Collection("commits").DeleteMany(context.Background(), bson.M{"project_id": project.ID})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete all branches for project
	_, err = config.MI.DB.Collection("branches").DeleteMany(context.Background(), bson.M{"project_id": project.ID})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete project
	_, err = config.MI.DB.Collection("projects").DeleteOne(context.Background(), bson.M{"_id": projectObjectId})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Project deleted",
	})
}
