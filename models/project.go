package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Project struct {
	ID              primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt       int64              `json:"created_at" bson:"created_at"`
	OwnerID         string             `json:"owner_id" bson:"owner_id"`
	Name            string             `json:"name" bson:"name" validate:"required"`
	DefaultBranchID primitive.ObjectID `json:"default_branch_id,omitempty" bson:"default_branch_id,omitempty"`
}
