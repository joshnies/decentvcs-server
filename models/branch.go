package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Branch struct {
	Id        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	Name      string             `json:"name" bson:"name" validate:"required"`
	ProjectId primitive.ObjectID `json:"project_id" bson:"project_id" validate:"required"`
	// TODO: Add user_id
}
