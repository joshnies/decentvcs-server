package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Branch struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	Name      string             `json:"name" bson:"name" validate:"required"`
	CommitID  primitive.ObjectID `json:"commit_id" bson:"commit_id" validate:"required"`
	// TODO: Add user_id
}

type BranchWithCommit struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	Name      string             `json:"name" bson:"name"`
	Commit    Commit             `json:"commit" bson:"commit"`
}

type BranchCreateDTO struct {
	Name        string `json:"name,omitempty"`
	CommitIndex int    `json:"commit_index,omitempty"`
}
