package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// [Database model]
//
// Team that owns projects.
type Team struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	// Owner's user ID.
	OwnerUserID string `json:"owner_user_id" bson:"owner_user_id"`
	// Team name. Must be unique (validated server-side).
	Name string `json:"name" bson:"name"`
}
