package acl

import (
	"context"
	"errors"
	"time"

	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Returns true if user has access to the given team (with any role).
func HasTeamAccess(userData models.UserData, teamName string, minRole models.Role) (models.HasTeamAccessResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get team
	var team models.Team
	if err := config.MI.DB.Collection("teams").FindOne(ctx, &bson.M{"name": teamName}).Decode(&team); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.HasTeamAccessResponse{HasAccess: false}, errors.New("team not found")
		}

		return models.HasTeamAccessResponse{HasAccess: false}, err
	}

	// Loop through roles
	for _, r := range userData.Roles {
		if r.TeamID.Hex() == team.ID.Hex() {
			if minRole != models.RoleNone {
				// Make sure user has access as minimum role or higher
				userRoleLvl, err := GetRoleLevel(r.Role)
				if err != nil {
					return models.HasTeamAccessResponse{HasAccess: false}, err
				}

				minRoleLvl, err := GetRoleLevel(minRole)
				if err != nil {
					return models.HasTeamAccessResponse{HasAccess: false}, err
				}

				if userRoleLvl >= minRoleLvl {
					// User has access with minimum role or higher
					return models.HasTeamAccessResponse{Team: &team, Role: r.Role, HasAccess: true}, nil
				}
			}

			// User has access to team
			return models.HasTeamAccessResponse{Team: &team, Role: models.RoleNone, HasAccess: true}, nil
		}
	}

	// No role found for team, so user has no access
	return models.HasTeamAccessResponse{HasAccess: false}, nil
}

// Get the numerical level of a role.
// Useful for comparing roles.
func GetRoleLevel(role models.Role) (uint, error) {
	switch role {
	case models.RoleCollab:
		return 1, nil
	case models.RoleAdmin:
		return 2, nil
	case models.RoleOwner:
		return 3, nil
	default:
		return 0, errors.New("invalid role")
	}
}

// Add a new role to a user's data in the VCS database.
func AddRole(userData models.UserData, teamID primitive.ObjectID, role models.Role) (models.UserData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add role
	userData.Roles = append(userData.Roles, models.RoleObject{
		Role:   role,
		TeamID: teamID,
	})

	// Update user data
	if _, err := config.MI.DB.Collection("user_data").UpdateOne(ctx, &bson.M{"user_id": userData.UserID}, &bson.M{"$set": &userData}); err != nil {
		return models.UserData{}, err
	}

	return userData, nil
}
