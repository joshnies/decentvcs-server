package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Project struct {
	ID                    primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt             int64              `json:"created_at" bson:"created_at"`
	Name                  string             `json:"name" bson:"name" validate:"required"`
	AccessGrant           string             `json:"access_grant,omitempty" bson:"access_grant,omitempty"`
	AccessGrantExpiration int64              `json:"access_grant_expiration,omitempty" bson:"access_grant_expiration,omitempty"`
	// TODO: Add owner_id
}
