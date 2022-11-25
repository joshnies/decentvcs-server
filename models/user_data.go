package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Data linked to a user in our auth provider.
type UserData struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	// ID of the user in our auth provider.
	UserID string `json:"user_id" bson:"user_id"`
	// Display name of the user.
	// By default, this is the prefix of the user's email address.
	Username string `json:"username" bson:"username"`
	// Roles for teams and projects.
	Roles []RoleObject `json:"roles" bson:"roles"`
	// ID of the team that new projects will be created in by default.
	DefaultTeamID primitive.ObjectID `json:"default_team_id" bson:"default_team_id"`
	// URL of the user's avatar.
	AvatarURL string `json:"avatar_url,omitempty" bson:"avatar_url,omitempty"`
	// Stripe customer ID.
	StripeCustomerID string `json:"stripe_customer_id,omitempty" bson:"stripe_customer_id,omitempty"`
	// ID of the current subscription this user has in Stripe.
	StripeSubscriptionID string `json:"stripe_subscription_id,omitempty" bson:"stripe_subscription_id,omitempty"`
}

// Request body for `UpdateUserData`.
type UpdateUserDataRequest struct {
	// Display name of the user.
	Username string `json:"username" bson:"username"`
	// URL of the user's avatar.
	// Required since it's currently the only updateable field.
	AvatarURL string `json:"avatar_url" validate:"required"`
}
