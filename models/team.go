package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// [Database model]
//
// Team that owns projects.
type Team struct {
	ID      primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name    string             `json:"name,omitempty" bson:"name,omitempty"`
	OwnerID primitive.ObjectID `json:"owner_id,omitempty" bson:"owner_id,omitempty"`
}
