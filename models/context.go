package models

type ContextKey string

const (
	ContextKeyStytchUser ContextKey = "stytch_user"
	ContextKeyUserData   ContextKey = "user_data"
	ContextKeyTeam       ContextKey = "team"
)
