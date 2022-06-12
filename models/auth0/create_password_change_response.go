package auth0

// Response body for "Create a password change ticket" endpoint.
//
// See Auth0 Management API docs:
// https://auth0.com/docs/api/management/v2#!/Tickets/post_password_change
type CreatePasswordChangeTicketResponse struct {
	Ticket string `json:"ticket"`
}
