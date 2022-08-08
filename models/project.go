package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Project struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	// Project name that must be unique in the scope of the team.
	Name string `json:"name" bson:"name" validate:"required"`
	// ID of the team that owns the project.
	TeamID          primitive.ObjectID `json:"team_id,omitempty" bson:"team_id,omitempty"`
	DefaultBranchID primitive.ObjectID `json:"default_branch_id,omitempty" bson:"default_branch_id,omitempty"`
	// URL of the backdrop image.
	BackdropURL string `json:"backdrop_url,omitempty" bson:"backdrop_url,omitempty"`
}

type UpdateProjectRequest struct {
	Name            string `json:"name"`
	DefaultBranchID string `json:"default_branch_id"`
	// URL of the backdrop image.
	BackdropURL string `json:"backdrop_url,omitempty" bson:"backdrop_url,omitempty"`
}

type InviteManyUsersDTO struct {
	Emails []string `json:"emails"`
}

type TransferProjectOwnershipRequest struct {
	NewTeamName string `json:"new_team_name"`
}
