package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Commit struct {
	ID            primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt     int64              `json:"created_at,omitempty" bson:"created_at,omitempty"`
	ProjectID     primitive.ObjectID `json:"project_id,omitempty" bson:"project_id,omitempty"`
	Message       string             `json:"message,omitempty" bson:"message,omitempty"`
	SnapshotPaths []string           `json:"snapshot_paths,omitempty" bson:"snapshot_paths,omitempty"`
	PatchPaths    []string           `json:"patch_paths,omitempty" bson:"patch_paths,omitempty"`
	DeletedPaths  []string           `json:"deleted_paths,omitempty" bson:"deleted_paths,omitempty"`
	HashMap       map[string]string  `json:"hash_map,omitempty" bson:"hash_map,omitempty"`
	// TODO: Add user ID
}
