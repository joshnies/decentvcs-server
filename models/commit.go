package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type CommitState struct {
	// Object key for file in storage
	FileKey string `json:"file_key"`
	// File hash used for determining changes
	Hash string `json:"hash"`
}

type Commit struct {
	ID               primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt        int64              `json:"created_at,omitempty" bson:"created_at,omitempty"`
	PreviousCommitID primitive.ObjectID `json:"previous_commit_id,omitempty" bson:"previous_commit_id,omitempty"`
	ProjectID        primitive.ObjectID `json:"project_id,omitempty" bson:"project_id,omitempty"`
	BranchID         primitive.ObjectID `json:"branch_id,omitempty" bson:"branch_id,omitempty"`
	Message          string             `json:"message,omitempty" bson:"message,omitempty"`
	// Array of fs paths to created files (uploaded as snapshots)
	CreatedFiles []string `json:"created_files,omitempty" bson:"created_files,omitempty"`
	// Array of fs paths to modified files (uploaded as snapshots)
	ModifiedFiles []string `json:"modified_files,omitempty" bson:"modified_files,omitempty"`
	// Array of fs paths to deleted files
	DeletedFiles []string `json:"deleted_files,omitempty" bson:"deleted_files,omitempty"`
	// Map of file path to hash
	HashMap map[string]string `json:"hash_map,omitempty" bson:"hash_map,omitempty"`
	// TODO: Add user ID
}

// Serialized version of Commit (ObjectID is replaced with string)
type CommitSerialized struct {
	ID               string `json:"_id,omitempty"`
	CreatedAt        int64  `json:"created_at,omitempty"`
	ProjectID        string `json:"project_id,omitempty"`
	BranchID         string `json:"branch_id,omitempty"`
	PreviousCommitID string `json:"previous_commit_id,omitempty"`
	Message          string `json:"message,omitempty"`
	// Array of fs paths to created files (uploaded as snapshots)
	CreatedFiles []string `json:"created_files,omitempty"`
	// Array of fs paths to modified files (uploaded as snapshots)
	ModifiedFiles []string `json:"modified_files,omitempty"`
	// Array of fs paths to deleted files
	DeletedFiles []string `json:"deleted_files,omitempty"`
	// Map of file path to hash
	HashMap map[string]string `json:"hash_map,omitempty"`
	// TODO: Add user ID
}
