package config

import (
	"log"
	"os"

	"github.com/stytchauth/stytch-go/v5/stytch"
	"github.com/stytchauth/stytch-go/v5/stytch/stytchapi"
)

var StytchClient *stytchapi.API

// Initialize stytch client.
func InitStytch() {
	projectID := os.Getenv("STYTCH_PROJECT_ID")
	if projectID == "" {
		log.Fatal("STYTCH_PROJECT_ID environment variable is not set.")
	}
	secret := os.Getenv("STYTCH_SECRET")
	if secret == "" {
		log.Fatal("STYTCH_SECRET environment variable is not set.")
	}

	stytchEnv := stytch.EnvTest
	if os.Getenv("STYTCH_LIVE") == "true" {
		stytchEnv = stytch.EnvLive
	}

	client, err := stytchapi.NewAPIClient(
		stytchEnv,
		projectID,
		secret,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Assign global instance
	StytchClient = client
}
