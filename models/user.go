package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Data linked to a user in our auth provider.
type UserData struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt int64              `json:"created_at,omitempty" bson:"created_at,omitempty"`
	UserID    string             `json:"user_id" bson:"user_id"`
	Roles     []RoleObject       `json:"roles" bson:"roles"`
	// ID of the team that new projects will be created in by default.
	DefaultTeamID primitive.ObjectID `json:"default_team_id" bson:"default_team_id"`
}
