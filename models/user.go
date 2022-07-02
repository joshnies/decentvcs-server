package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Data linked to a user in our auth provider.
type UserData struct {
	ID     primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	UserID string             `json:"user_id" bson:"user_id"`
	Roles  []RoleObject       `json:"roles" bson:"roles"`
}
