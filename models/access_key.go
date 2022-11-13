package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AccessKey struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	ExpiresAt time.Time          `json:"expires_at" bson:"expires_at"`
	// ID of the user in our auth provider.
	UserID string `json:"user_id" bson:"user_id"`
	// ID of the team that owns the project.
	TeamID primitive.ObjectID `json:"team_id,omitempty" bson:"team_id,omitempty"`
	Scope  string             `json:"scope" bson:"scope"`
}
