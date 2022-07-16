package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Role string

const (
	RoleNone   Role = ""
	RoleCollab Role = "collab"
	RoleAdmin  Role = "admin"
	RoleOwner  Role = "owner"
)

type RoleObject struct {
	Role   Role               `json:"role" bson:"role"`
	TeamID primitive.ObjectID `json:"team_id,omitempty" bson:"team_id,omitempty"`
}
