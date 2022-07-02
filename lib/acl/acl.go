package acl

import (
	"context"
	"errors"
	"time"

	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Returns true if user has access to the given project (with any role).
func HasProjectAccess(userID string, projectID string) (bool, error) {
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

	// Loop through roles
	for _, r := range userData.Roles {
		if r.ProjectID.Hex() == projectID {
			// User has access to project
			return true, nil
		}
	}

	// No role found for project, so user has no access
	return false, nil
}

// Returns user's role, if any, for the given project.
// If no role is found, returns -1.
func GetProjectRole(userID string, projectID string) (models.Role, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user data
	var userData models.UserData
	if err := config.MI.DB.Collection("user_data").FindOne(ctx, &bson.M{"user_id": userID}).Decode(&userData); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", errors.New("user not found")
		}

		return "", err
	}

	// Loop through roles
	for _, robj := range userData.Roles {
		if robj.ProjectID.Hex() == projectID {
			// User has a role for project
			return robj.Role, nil
		}
	}

	// No role found for project
	return "", nil
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
