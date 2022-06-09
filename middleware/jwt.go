package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/joshnies/decent-vcs-api/config"
)

type CustomClaims struct {
	Scope string `json:"scope"`
}

// Validate claims
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// Middleware that checks for JWT validity
//
// NOTE: This is standard http middleware, not fiber middleware due to `jwtmiddleware` limitations
//
func ValidateJWT() func(next http.Handler) http.Handler {
	provider := jwks.NewCachingProvider(config.I.Auth0.IssuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		config.I.Auth0.IssuerURL.String(),
		[]string{config.I.Auth0.Audience},
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to set up JWT validator")
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Encountered error while validating JWT: %v", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Failed to validate JWT."}`))
	}

	middleware := jwtmiddleware.New(
		jwtValidator.ValidateToken,
		jwtmiddleware.WithErrorHandler(errorHandler),
	)

	return func(next http.Handler) http.Handler {
		return middleware.CheckJWT(next)
	}
}
