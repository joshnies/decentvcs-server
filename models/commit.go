package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Commit struct {
	Id        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	Message   string             `json:"message" bson:"message" validate:"required"`
	BranchId  primitive.ObjectID `json:"branch_id" bson:"branch_id" validate:"required"`
	// TODO: Add user_id
}
