package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Commit struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	Index     int                `json:"index,omitempty" bson:"index,omitempty"`
	ProjectID primitive.ObjectID `json:"project_id,omitempty" bson:"project_id,omitempty"`
	BranchID  primitive.ObjectID `json:"branch_id,omitempty" bson:"branch_id,omitempty"`
	Message   string             `json:"message,omitempty" bson:"message,omitempty"`
	// Array of fs paths to created files (uploaded as snapshots)
	CreatedFiles []string `json:"created_files,omitempty" bson:"created_files,omitempty"`
	// Array of fs paths to modified files (uploaded as snapshots)
	ModifiedFiles []string `json:"modified_files,omitempty" bson:"modified_files,omitempty"`
	// Array of fs paths to deleted files
	DeletedFiles []string `json:"deleted_files,omitempty" bson:"deleted_files,omitempty"`
	// Map of file path to hash
	HashMap map[string]string `json:"hash_map,omitempty" bson:"hash_map,omitempty"`
	// ID of the user who made the commit.
	// If empty, then the system created it.
	AuthorID string `json:"author_id,omitempty" bson:"author_id,omitempty"`
}

// Serialized version of Commit (ObjectID is replaced with string)
type CreateCommitRequest struct {
	BranchID string `json:"branch_id" validate:"required"`
	Message  string `json:"message"`
	// Array of fs paths to created files (uploaded as snapshots)
	CreatedFiles []string `json:"created_files"`
	// Array of fs paths to modified files (uploaded as snapshots)
	ModifiedFiles []string `json:"modified_files"`
	// Array of fs paths to deleted files
	DeletedFiles []string `json:"deleted_files"`
	// Map of file path to hash
	HashMap map[string]string `json:"hash_map" validate:"required"`
	// ID of the user who made the commit.
	// If empty, then the system created it.
	AuthorID string `json:"author_id"`
}

type CommitWithBranch struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	Index     int                `json:"index,omitempty" bson:"index,omitempty"`
	ProjectID primitive.ObjectID `json:"project_id,omitempty" bson:"project_id,omitempty"`
	Branch    Branch             `json:"branch,omitempty" bson:"branch,omitempty"`
	Message   string             `json:"message,omitempty" bson:"message,omitempty"`
	// Array of fs paths to created files (uploaded as snapshots)
	CreatedFiles []string `json:"created_files,omitempty" bson:"created_files,omitempty"`
	// Array of fs paths to modified files (uploaded as snapshots)
	ModifiedFiles []string `json:"modified_files,omitempty" bson:"modified_files,omitempty"`
	// Array of fs paths to deleted files
	DeletedFiles []string `json:"deleted_files,omitempty" bson:"deleted_files,omitempty"`
	// Map of file path to hash
	HashMap map[string]string `json:"hash_map,omitempty" bson:"hash_map,omitempty"`
	// ID of the user who made the commit.
	// If empty, then the system created it.
	AuthorID string `json:"author_id,omitempty" bson:"author_id,omitempty"`
}
