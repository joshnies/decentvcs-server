package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Project struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	// Project name that must be unique in the scope of the team.
	Name string `json:"name" bson:"name" validate:"required"`
	// ID of the team that owns the project.
	TeamID          primitive.ObjectID `json:"team_id" bson:"team_id"`
	DefaultBranchID primitive.ObjectID `json:"default_branch_id" bson:"default_branch_id"`
	// URL of the thumbnail image.
	ThumbnailURL string `json:"thumbnail_url,omitempty" bson:"thumbnail_url,omitempty"`
	// If `true`, modified committed files in this project will be uploaded as patches instead of snapshots (e.g. the
	// whole file).
	EnablePatchRevisions bool `json:"enable_patch_revisions" bson:"enable_patch_revisions"`
}

type CreateProjectRequest struct {
	// URL of the thumbnail image.
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	// If `true`, modified committed files in this project will be uploaded as patches instead of snapshots (e.g. the
	// whole file).
	EnablePatchRevisions bool `json:"enable_patch_revisions,omitempty"`
}

type UpdateProjectRequest struct {
	Name            string `json:"name"`
	DefaultBranchID string `json:"default_branch_id"`
	// URL of the thumbnail image.
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	// If `true`, modified committed files in this project will be uploaded as patches instead of snapshots (e.g. the
	// whole file).
	// EnablePatchRevisions bool `json:"enable_patch_revisions,omitempty"`
}

type InviteManyUsersDTO struct {
	Emails []string `json:"emails"`
}

type TransferProjectOwnershipRequest struct {
	NewTeamName string `json:"new_team_name"`
}
