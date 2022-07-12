package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Plan string

const (
	PlanTrial      Plan = "trial"
	PlanCloud      Plan = "cloud"
	PlanEnterprise Plan = "enterprise"
)

type TeamBilling struct {
	// Plan that this team subscribes to.
	Plan Plan `json:"plan" bson:"plan"`
	// Unix timestamp of when the billing period started.
	PeriodStart int64 `json:"period_start" bson:"period_start"`
	// Amount of storage used in MB. Accounts for all projects within this team.
	StorageUsedMB int64 `json:"storage_used_mb" bson:"storage_used_mb"`
	// Amount of bandwidth used in MB.  Accounts for all projects within this team.
	// Resets on the first day of a new billing period.
	BandwidthUsedMB int64 `json:"bandwidth_used_mb" bson:"bandwidth_used_mb"`
}

// [Database model]
//
// Team that owns projects.
type Team struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	// Team name. Must be unique (validated server-side).
	Name    string      `json:"name" bson:"name"`
	Billing TeamBilling `json:"billing" bson:"billing"`
}

// Request body for `CreateOneTeam` or `UpdateOneTeam`.
type CreateOrUpdateTeamRequest struct {
	// Team name. Must be unique (validated server-side).
	Name string `json:"name" validate:"required,min=3,max=64"`
}
