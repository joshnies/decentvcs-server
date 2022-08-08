package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Plan string

const (
	PlanTrial      Plan = "trial"
	PlanCloud      Plan = "cloud"
	PlanEnterprise Plan = "enterprise"
)

// [Database model]
//
// Team that owns projects.
type Team struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	// Team name. Must be unique (validated server-side).
	Name string `json:"name" bson:"name"`
	// Plan that this team subscribes to.
	Plan Plan `json:"plan" bson:"plan"`
	// Unix timestamp of when the billing period started.
	PeriodStart time.Time `json:"period_start" bson:"period_start"`
	// Amount of storage used in MB. Accounts for all projects within this team.
	StorageUsedMB float64 `json:"storage_used_mb" bson:"storage_used_mb"`
	// Amount of bandwidth used in MB.  Accounts for all projects within this team.
	// Resets on the first day of a new billing period.
	BandwidthUsedMB float64 `json:"bandwidth_used_mb" bson:"bandwidth_used_mb"`
	// URL of the team's backdrop image.
	BackdropURL string `json:"backdrop_url,omitempty" bson:"backdrop_url,omitempty"`
}

// Request body for `CreateOneTeam`.
type CreateTeamRequest struct {
	// Team name. Must be unique (validated server-side).
	Name string `json:"name" validate:"required,min=3,max=64"`
	// Plan that this team subscribes to.
	Plan Plan `json:"plan"`
	// Unix timestamp of when the billing period started.
	PeriodStart time.Time `json:"period_start"`
	// URL of the team's backdrop image.
	BackdropURL string `json:"backdrop_url"`
}

// Request body for `UpdateOneTeam`.
type UpdateTeamRequest struct {
	// Team name. Must be unique (validated server-side).
	// Required since it's currently the only updateable field.
	Name string `json:"name" validate:"min=3,max=64"`
	// Amount of storage used in MB. Accounts for all projects within this team.
	// Provide -1 to reset to 0.
	StorageUsedMB float64 `json:"storage_used_mb" validate:"gte=0"`
	// Amount of bandwidth used in MB.  Accounts for all projects within this team.
	// Resets on the first day of a new billing period.
	// Provide -1 to reset to 0.
	BandwidthUsedMB float64 `json:"bandwidth_used_mb" validate:"gte=0"`
	// URL of the team's backdrop image.
	BackdropURL string `json:"backdrop_url,omitempty"`
}

// Request body for `UpdateOneTeamPlan`.
type UpdateTeamPlanRequest struct {
	// Plan that this team subscribes to.
	Plan Plan `json:"plan" validate:"min=1"`
}
