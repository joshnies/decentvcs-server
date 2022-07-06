package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Branch struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	DeletedAt int64              `json:"deleted_at" bson:"deleted_at"`
	Name      string             `json:"name" bson:"name" validate:"required"`
	ProjectID primitive.ObjectID `json:"project_id" bson:"project_id" validate:"required"`
	// ID of the commit that this branch currently points to (a.k.a. the latest commit).
	CommitID primitive.ObjectID `json:"commit_id" bson:"commit_id" validate:"required"`
	// Map of file path to user ID.
	// Denotes which file paths are currently locked for this branch, and by whom.
	Locks map[string]string `json:"locks" bson:"locks"`
}

type BranchWithCommit struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	DeletedAt int64              `json:"deleted_at" bson:"deleted_at"`
	Name      string             `json:"name" bson:"name"`
	ProjectID primitive.ObjectID `json:"project_id" bson:"project_id"`
	// The commit that this branch currently points to (a.k.a. the latest commit).
	Commit Commit `json:"commit" bson:"commit"`
	// Map of file path to user ID.
	// Denotes which file paths are currently locked for this branch, and by whom.
	Locks map[string]string `json:"locks" bson:"locks"`
}

type BranchCreateDTO struct {
	Name        string `json:"name,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	CommitIndex int    `json:"commit_index,omitempty"`
}

type BranchCreateBSON struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	Name      string             `json:"name" bson:"name"`
	ProjectID primitive.ObjectID `json:"project_id" bson:"project_id"`
	// The commit that this branch currently points to (a.k.a. the latest commit).
	CommitID primitive.ObjectID `json:"commit_id" bson:"commit_id"`
	// Map of file path to user ID.
	// Denotes which file paths are currently locked for this branch, and by whom.
	Locks map[string]string `json:"locks" bson:"locks"`
}

type BranchUpdateDTO struct {
	Name string `json:"name,omitempty"`
}
