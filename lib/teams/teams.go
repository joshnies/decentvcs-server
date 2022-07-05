package teams

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Create the default team for a new user.
func CreateDefault(userID string, email string) (models.Team, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create default team
	emailUser := strings.Split(email, "@")[0]

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
		teamName += "-" + strings.Replace(strings.Replace(userID, "user-", "", 1), "test-", "", 1)
	}

	team := models.Team{
		ID:          primitive.NewObjectID(),
		CreatedAt:   time.Now().Unix(),
		OwnerUserID: userID,
		Name:        teamName,
	}
	if _, err := config.MI.DB.Collection("teams").InsertOne(ctx, team); err != nil {
		fmt.Printf("Error fetching default team while authenticating user with ID \"%s\": %v\n", userID, err)
		return models.Team{}, err
	}

	return team, nil
}
