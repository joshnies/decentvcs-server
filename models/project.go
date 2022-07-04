package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Project struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	// Project name that must be unique in the scope of the team.
	Name string `json:"name" bson:"name" validate:"required"`
	// Project name prefixed by the team name. For example: `my-team/my-project`.
	// This is saved to the database for easy access, and must be updated whenever the project name or team name
	// changes.
	Blob string `json:"blob" bson:"blob"`
	// Team that owns the project.
	TeamID          primitive.ObjectID `json:"team_id,omitempty" bson:"team_id,omitempty"`
	DefaultBranchID primitive.ObjectID `json:"default_branch_id,omitempty" bson:"default_branch_id,omitempty"`
}

type InviteManyUsersDTO struct {
	Emails []string `json:"emails"`
}
