package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/config"
	"github.com/joshnies/qc-api/lib/auth"
	"github.com/joshnies/qc-api/lib/storage"
	"github.com/joshnies/qc-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

// Get one project by ID.
//
// URL params:
//
// - id: Project ID
//
func GetOneProject(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objId, _ := primitive.ObjectIDFromHex(c.Params("pid"))

	// Get project from database
	var result models.Project
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

// Get one project by blob.
//
// URL params:
//
// - oa: Alias of the user or team who owns the project
//
// - pname: Name of the project
//
func GetOneProjectByBlob(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ownerAlias := c.Params("oa")
	projectName := c.Params("pname")

	// Get project from database
	var result models.Project
	err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"name": projectName}).Decode(&result)
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

	// Get user from context
	sub, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

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

	// Generate default branch ID ahead of time
	branchId := primitive.NewObjectID()

	// Create new project
	project := models.Project{
		ID:              primitive.NewObjectID(),
		CreatedAt:       time.Now().Unix(),
		OwnerID:         sub,
		Name:            body.Name,
		DefaultBranchID: branchId,
	}

	// Create project in database
	_, err = config.MI.DB.Collection("projects").InsertOne(ctx, project)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create initial commit
	commit := models.Commit{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().Unix(),
		Index:     1, // Starts at 1 since in Go, 0 is the default and used to check for empty values
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
	branch := models.BranchCreateBSON{
		ID:        branchId,
		CreatedAt: time.Now().Unix(),
		Name:      "stable",
		ProjectID: project.ID,
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
				"commit": fiber.Map{
					"_id":   commit.ID.Hex(),
					"index": commit.Index,
				},
			},
		},
	})
}

// Update a project.
//
// URL params:
//
// - pid: Project ID
//
// Body: (any field from Project)
//
// Returns the updated project.
func UpdateOneProject(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse body
	var body models.Project
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Parse project ID
	projectObjectId, err := primitive.ObjectIDFromHex(c.Params("pid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Make sure user is project owner
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get project to make sure user is owner
	var project models.Project
	err = config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"owner_id": userId}).Decode(&project)
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

	updateData := bson.M{}
	if body.OwnerID != "" {
		updateData["owner_id"] = body.OwnerID
	}
	if body.Name != "" {
		updateData["name"] = body.Name
	}
	if body.DefaultBranchID != primitive.NilObjectID {
		updateData["default_branch_id"] = body.DefaultBranchID
	}

	// Update project
	_, err = config.MI.DB.Collection("projects").UpdateOne(
		ctx,
		bson.M{"_id": projectObjectId},
		bson.M{"$set": updateData},
	)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"_id":               projectObjectId.Hex(),
		"owner_id":          body.OwnerID,
		"name":              body.Name,
		"default_branch_id": body.DefaultBranchID.Hex(),
	})
}

// Delete project and all of its subresources.
func DeleteOneProject(c *fiber.Ctx) error {
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

// Create many presigned URLs for storage, scoped to a project.
//
// URL params:
//
// - pid: Project ID
//
// Body:
//
// - keys: Array of object keys
//
// Returns an array of presigned URLs.
//
func CreatePresignedURLs(c *fiber.Ctx) error {
	// Get user ID
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse project ID
	pid := c.Params("pid")
	projectObjectId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	project := models.Project{}
	err = config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"_id": projectObjectId, "owner_id": userId}).Decode(&project)
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

	// Parse request body
	var body models.PresignedURLRequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Get presigned URLs
	urls, err := storage.GetManyPresignedURLs(pid, body.Keys)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(urls)
}
