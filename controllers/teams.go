package controllers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/auth"
	"github.com/joshnies/decent-vcs/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Get one team.
func GetOneTeam(c *fiber.Ctx) error {
	// Get team ID
	teamID, err := primitive.ObjectIDFromHex(c.Params("tid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid team ID",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get team
	var team models.Team
	if err := config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"_id": teamID}).Decode(&team); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Not found",
			"message": "Team not found",
		})
	}

	return c.JSON(team)
}

// Get many teams.
func GetManyTeams(c *fiber.Ctx) error {
	// Get pagination query parameters
	var skip int64 = 0
	var limit int64 = 25

	if c.Query("skip") != "" {
		skip, _ = strconv.ParseInt(c.Query("skip"), 10, 64)
	}

	if c.Query("limit") != "" {
		limit, _ = strconv.ParseInt(c.Query("limit"), 10, 64)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all teams (paginated)
	var teams []models.Team
	cur, err := config.MI.DB.Collection("teams").Find(ctx, bson.M{}, &options.FindOptions{
		Skip:  &skip,
		Limit: &limit,
	})
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Not found",
			"message": "Team not found",
		})
	}

	cur.All(ctx, &teams)

	return c.JSON(teams)
}

// Create a new team.
func CreateTeam(c *fiber.Ctx) error {
	// Get user ID
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse request body
	var reqBody models.CreateOrUpdateTeamRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid request body",
		})
	}

	// Validate request body
	if err := config.Validator.Struct(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if team name is unique
	var existingTeam models.Team
	err = config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"name": reqBody.Name}).Decode(&existingTeam)
	if err == nil || !errors.Is(err, mongo.ErrNoDocuments) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "A team with that name already exists",
		})
	}
	if err != nil {
		fmt.Printf("Error checking if team name is unique: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create team
	teamID := primitive.NewObjectID()
	team := models.Team{
		ID:        teamID,
		CreatedAt: time.Now().Unix(),
		Name:      reqBody.Name,
	}

	if _, err := config.MI.DB.Collection("teams").InsertOne(ctx, team); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal server error",
			"message": "Failed to create team",
		})
	}

	// Add team owner role to user
	if _, err := config.MI.DB.Collection("users").UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$push": bson.M{"roles": models.RoleObject{
		Role:   models.RoleOwner,
		TeamID: teamID,
	}}}); err != nil {
		fmt.Printf("Error adding team owner role to user: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(team)
}

// Update a team.
// Only team admins can update a team.
func UpdateTeam(c *fiber.Ctx) error {
	// Get team ID
	teamID, err := primitive.ObjectIDFromHex(c.Params("tid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid team ID",
		})
	}

	// Parse request body
	var reqBody models.CreateOrUpdateTeamRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid request body",
		})
	}

	// Validate request body
	if err := config.Validator.Struct(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if team name is unique
	var existingTeam models.Team
	err = config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"name": reqBody.Name}).Decode(&existingTeam)
	if err == nil || !errors.Is(err, mongo.ErrNoDocuments) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "A team with that name already exists",
		})
	}
	if err != nil {
		fmt.Printf("Error checking if team name is unique: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get team
	var team models.Team
	if err := config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"_id": teamID}).Decode(&team); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "Not found",
				"message": "Team not found",
			})
		}

		fmt.Printf("Error getting team: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Update team
	if _, err := config.MI.DB.Collection("teams").UpdateOne(ctx, bson.M{"_id": teamID}, bson.M{"$set": bson.M{
		"name": reqBody.Name,
	}}); err != nil {
		fmt.Printf("Error updating team: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	team.Name = reqBody.Name
	return c.JSON(team)
}

// Delete a team.
// Only team owners can delete a team.
// A user's default team cannot be deleted.
func DeleteTeam(c *fiber.Ctx) error {
	// Get user ID
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get team ID
	teamID, err := primitive.ObjectIDFromHex(c.Params("tid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid team ID",
		})
	}

	// Parse request body
	var reqBody models.CreateOrUpdateTeamRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid request body",
		})
	}

	// Validate request body
	if err := config.Validator.Struct(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get team to make sure it exists
	var team models.Team
	if err := config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"_id": teamID}).Decode(&team); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "Not found",
				"message": "Team not found",
			})
		}

		fmt.Printf("Error getting team: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": userID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			fmt.Printf("User data not found for user with ID \"%s\"", userID)
		} else {
			fmt.Printf("Error getting user data: %v\n", err)
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Ensure deleting team is not the default for the user
	if team.ID.Hex() == userData.DefaultTeamID.Hex() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Cannot delete default team",
		})
	}

	// Delete team
	if _, err := config.MI.DB.Collection("teams").DeleteOne(ctx, bson.M{"_id": teamID}); err != nil {
		fmt.Printf("Error deleting team: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Team deleted successfully",
	})
}
