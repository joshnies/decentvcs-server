package auth0

type User struct {
	UserID        string                 `json:"user_id"`
	Email         string                 `json:"email"`
	EmailVerified bool                   `json:"email_verified"`
	Username      string                 `json:"username"`
	PhoneNumber   string                 `json:"phone_number"`
	PhoneVerified bool                   `json:"phone_verified"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
	Identities    []Identity             `json:"identities"`
	AppMetadata   map[string]interface{} `json:"app_metadata"`
	UserMetadata  map[string]interface{} `json:"user_metadata"`
	Picture       string                 `json:"picture"`
	Name          string                 `json:"name"`
	Nickname      string                 `json:"nickname"`
	Multifactor   []string               `json:"multifactor"`
	LastIP        string                 `json:"last_ip"`
	LastLogin     string                 `json:"last_login"`
	LoginsCount   int                    `json:"logins_count"`
	Blocked       bool                   `json:"blocked"`
	GivenName     string                 `json:"given_name"`
	FamilyName    string                 `json:"family_name"`
}
