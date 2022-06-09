package auth

import (
	"encoding/json"
	"fmt"

	"github.com/go-jose/go-jose"
	"github.com/gofiber/fiber/v2"
)

// Get the user subject string (sub) from the decoded access token
func GetUserID(c *fiber.Ctx) (string, error) {
	// Get JWT from "Authorization" header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return "", fiber.ErrUnauthorized
	}

	// Trim "Bearer " prefix
	authHeader = authHeader[len("Bearer "):]

	// Decode JWT
	token, err := jose.ParseSigned(authHeader)
	if err != nil {
		fmt.Printf("Failed to parse JWT while getting sub claim: %s\n", err)
		return "", fiber.ErrUnauthorized
	}

	// Get payload and convert to JSON
	// No verification is needed since the JWT is already validated via Auth0's JWT middleware
	payload := token.UnsafePayloadWithoutVerification()
	var claims map[string]interface{}
	err = json.Unmarshal(payload, &claims)
	if err != nil {
		fmt.Printf("Failed to unmarshal JWT payload while getting sub claim: %s\n", err)
		return "", fiber.ErrUnauthorized
	}

	// Return sub claim if it exists, otherwise return error
	if sub, ok := claims["sub"].(string); ok {
		return sub, nil
	}

	fmt.Println("No sub claim found in JWT payload")
	return "", fiber.ErrUnauthorized
}
