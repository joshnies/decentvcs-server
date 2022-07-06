package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Branch struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id"`
	CreatedAt   int64              `json:"created_at" bson:"created_at"`
	DeletedAt   int64              `json:"deleted_at" bson:"deleted_at"`
	Name        string             `json:"name" bson:"name" validate:"required"`
	ProjectID   primitive.ObjectID `json:"project_id" bson:"project_id" validate:"required"`
	CommitID    primitive.ObjectID `json:"commit_id" bson:"commit_id" validate:"required"`
	LockedPaths []string           `json:"locked_paths" bson:"locked_paths"`
}

type BranchWithCommit struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id"`
	CreatedAt   int64              `json:"created_at" bson:"created_at"`
	DeletedAt   int64              `json:"deleted_at" bson:"deleted_at"`
	Name        string             `json:"name" bson:"name"`
	ProjectID   primitive.ObjectID `json:"project_id" bson:"project_id"`
	Commit      Commit             `json:"commit" bson:"commit"`
	LockedPaths []string           `json:"locked_paths" bson:"locked_paths"`
}

type BranchCreateDTO struct {
	Name        string `json:"name,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	CommitIndex int    `json:"commit_index,omitempty"`
}

type BranchCreateBSON struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	Name      string             `json:"name" bson:"name" validate:"required"`
	ProjectID primitive.ObjectID `json:"project_id" bson:"project_id" validate:"required"`
	CommitID  primitive.ObjectID `json:"commit_id" bson:"commit_id" validate:"required"`
}

type BranchUpdateDTO struct {
	Name string `json:"name,omitempty"`
}
