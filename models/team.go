package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// [Database model]
//
// Team that owns projects.
type Team struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	// Team name. Must be unique (validated server-side).
	Name string `json:"name" bson:"name"`
}

// Request body for `CreateOneTeam`.
type CreateTeamRequest struct {
	// Team name. Must be unique (validated server-side).
	Name string `json:"name" validate:"required,min=3,max=64"`
}
