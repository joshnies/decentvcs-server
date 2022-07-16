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
func HasTeamAccess(userID string, teamName string, minRole models.Role) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, &bson.M{"user_id": userID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, errors.New("user not found")
		}

		return false, err
	}

	// Get team
	var team models.Team
	if err := config.MI.DB.Collection("team").FindOne(ctx, &bson.M{"name": teamName}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, errors.New("team not found")
		}

		return false, err
	}

	// Loop through roles
	for _, r := range userData.Roles {
		if r.TeamID.Hex() == team.ID.Hex() {
			if minRole != models.RoleNone {
				// Make sure user has access as minimum role or higher
				userRoleLvl, err := GetRoleLevel(r.Role)
				if err != nil {
					return false, err
				}

				minRoleLvl, err := GetRoleLevel(minRole)
				if err != nil {
					return false, err
				}

				return userRoleLvl >= minRoleLvl, nil
			}

			// User has access to team
			return true, nil
		}
	}

	// No role found for team, so user has no access
	return false, nil
}

// Returns user's role, if any, for the given team.
// If no role is found, returns `models.RoleNone`.
func GetTeamRole(userID string, teamName string) (models.Role, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, &bson.M{"user_id": userID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.RoleNone, errors.New("user not found")
		}

		return models.RoleNone, err
	}

	// Get team
	var team models.Team
	if err := config.MI.DB.Collection("team").FindOne(ctx, &bson.M{"name": teamName}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.RoleNone, errors.New("team not found")
		}

		return models.RoleNone, err
	}

	// Loop through roles
	for _, robj := range userData.Roles {
		if robj.TeamID.Hex() == team.ID.Hex() {
			// User has a role for team
			return robj.Role, nil
		}
	}

	// No role found for team
	return models.RoleNone, nil
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
func AddRole(userID string, projectID primitive.ObjectID, role models.Role) (models.UserData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, &bson.M{"user_id": userID}).Decode(&userData); err != nil {
		return models.UserData{}, err
	}

	// Add role
	userData.Roles = append(userData.Roles, models.RoleObject{
		ProjectID: projectID,
		Role:      role,
	})

	// Update user data
	if _, err := config.MI.DB.Collection("user_data").UpdateOne(ctx, &bson.M{"user_id": userID}, &bson.M{"$set": &userData}); err != nil {
		return models.UserData{}, err
	}

	return userData, nil
}
