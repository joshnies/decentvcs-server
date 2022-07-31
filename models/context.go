package models

type ContextKey string

const (
	ContextKeyStytchUser ContextKey = "stytch_user"
	ContextKeyUser       ContextKey = "user"
	ContextKeyTeam       ContextKey = "team"
)
