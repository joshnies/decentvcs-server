package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Role string

const (
	RoleCollab Role = "collab"
	RoleAdmin  Role = "admin"
	RoleOwner  Role = "owner"
)

type RoleObject struct {
	ProjectID primitive.ObjectID `json:"project_id" bson:"project_id"`
	Role      Role               `json:"role" bson:"role"`
}
