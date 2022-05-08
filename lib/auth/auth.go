package auth

import (
	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gofiber/fiber/v2"
)

// Get the user subject string (sub) from the decoded access token
func GetUserSub(c *fiber.Ctx) string {
	token := c.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	return token.RegisteredClaims.Subject
}
