package team_lib

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Get team from user context.
func GetTeamFromContext(c *fiber.Ctx) *models.Team {
	return c.UserContext().Value(models.ContextKeyTeam).(*models.Team)
}

// Create the default team for a new user.
func CreateDefault(userID string, email string) (models.Team, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create default team
	emailUser := strings.Split(email, "@")[0]

	// Get user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, bson.M{"user_id": userID}).Decode(&userData); err != nil {
		if err == mongo.ErrNoDocuments {
			return models.Team{}, errors.New("user data not found")
		}
		return models.Team{}, err
	}

	// Check if there's a team already with that name
	alreadyExists := true
	var existingTeam models.Team
	if err := config.MI.DB.Collection("teams").FindOne(ctx, bson.M{"name": emailUser}).Decode(&existingTeam); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			alreadyExists = false
		} else {
			fmt.Printf("Error searching for existing team with name \"%s\": %v\n", emailUser, err)
			return models.Team{}, err
		}
	}

	teamName := emailUser
	if alreadyExists {
		teamName += "-" + strings.Replace(strings.Replace(userData.UserID, "user-", "", 1), "test-", "", 1)
	}

	// Create team
	team := models.Team{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now(),
		Name:      teamName,
	}
	if _, err := config.MI.DB.Collection("teams").InsertOne(ctx, team); err != nil {
		fmt.Printf("Error creating default team for user with ID \"%s\": %v\n", userData.UserID, err)
		return models.Team{}, err
	}

	// Update user data with roles and set new team as default
	roles := userData.Roles
	roles = append(roles, models.RoleObject{
		Role:   models.RoleOwner,
		TeamID: team.ID,
	})
	if _, err := config.MI.DB.Collection("user_data").UpdateOne(ctx, bson.M{"_id": userData.ID}, bson.M{"$set": bson.M{"roles": roles, "default_team_id": team.ID}}); err != nil {
		// Delete created team
		if _, err := config.MI.DB.Collection("teams").DeleteOne(ctx, bson.M{"_id": team.ID}); err != nil {
			fmt.Printf("Error deleting team after failing to create owner role: %v\n", err)
		}

		fmt.Printf("Error adding team owner role to user: %v\n", err)
		return models.Team{}, err
	}

	return team, nil
}
