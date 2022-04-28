package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Commit struct {
	ID               primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt        int64              `json:"created_at,omitempty" bson:"created_at,omitempty"`
	PreviousCommitID primitive.ObjectID `json:"previous_commit_id,omitempty" bson:"previous_commit_id,omitempty"`
	BranchID         primitive.ObjectID `json:"branch_id,omitempty" bson:"branch_id,omitempty"`
	Message          string             `json:"message,omitempty" bson:"message,omitempty"`
	SnapshotPaths    []string           `json:"snapshot_paths,omitempty" bson:"snapshot_paths,omitempty"`
	PatchPaths       []string           `json:"patch_paths,omitempty" bson:"patch_paths,omitempty"`
	DeletedPaths     []string           `json:"deleted_paths,omitempty" bson:"deleted_paths,omitempty"`
	HashMap          map[string]string  `json:"hash_map,omitempty" bson:"hash_map,omitempty"`
	// TODO: Add user ID
}

// Serialized version of Commit (ObjectID is replaced with string)
type CommitSerialized struct {
	ID               string            `json:"_id,omitempty"`
	CreatedAt        int64             `json:"created_at,omitempty"`
	BranchID         string            `json:"branch_id,omitempty"`
	PreviousCommitID string            `json:"previous_commit_id,omitempty"`
	Message          string            `json:"message,omitempty"`
	SnapshotPaths    []string          `json:"snapshot_paths,omitempty"`
	PatchPaths       []string          `json:"patch_paths,omitempty"`
	DeletedPaths     []string          `json:"deleted_paths,omitempty"`
	HashMap          map[string]string `json:"hash_map,omitempty"`
}
