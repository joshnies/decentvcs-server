package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Commit struct {
	Id        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt int64              `json:"created_at,omitempty" bson:"created_at,omitempty"`
	Message   string             `json:"message,omitempty" bson:"message,omitempty" validate:"required"`
	BranchId  primitive.ObjectID `json:"branch_id,omitempty" bson:"branch_id,omitempty"`
	FileURI   string             `json:"file_uri,omitempty" bson:"file_uri,omitempty"`
	// TODO: Add user_id
}
