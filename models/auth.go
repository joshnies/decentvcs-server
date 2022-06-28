package models

// Request body for `/authenticate`
type AuthenticateRequest struct {
	Token        string `json:"token" validate:"required"`
	SessionToken string `json:"session_token"`
}

// Response body for `/authenticate`
type AuthenticateResponse struct {
	SessionToken string `json:"session_token"`
}
