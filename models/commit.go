package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Commit struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt int64              `json:"created_at,omitempty" bson:"created_at,omitempty"`
	ProjectID primitive.ObjectID `json:"project_id,omitempty" bson:"project_id,omitempty"`
	Message   string             `json:"message,omitempty" bson:"message,omitempty"`
	FileURIs  []string           `json:"file_uris,omitempty" bson:"file_uris,omitempty"`
	// TODO: Add user_id
}
