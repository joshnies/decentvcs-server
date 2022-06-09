package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/go-co-op/gocron"
)

type Auth0Config struct {
	ClientID     string
	ClientSecret string
	Domain       string
	Audience     string
	IssuerURL    *url.URL
	// Management API access token
	ManagementToken string
	// Management API audience
	ManagementAudience string
}

type Config struct {
	Debug          bool
	Port           string
	Scheduler      *gocron.Scheduler
	MaxInviteCount int
	Auth0          Auth0Config
}

// Global config instance
var I Config

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}

// Initialize global config instance
// NOTE: This should only ever be called once (at the start of the app)
func InitConfig() {
	// Global
	maxInviteCountStr := os.Getenv("MAX_INVITE_COUNT")
	if maxInviteCountStr == "" {
		maxInviteCountStr = "10"
	}
	maxInviteCount, err := strconv.Atoi(maxInviteCountStr)
	if err != nil {
		log.Fatal("MAX_INVITE_COUNT must be an integer")
	}
	if maxInviteCount <= 0 {
		log.Fatal("MAX_INVITE_COUNT must be greater than 0")
	}

	// Auth0
	auth0ClientID := os.Getenv("AUTH0_CLIENT_ID")
	if auth0ClientID == "" {
		panic("AUTH0_CLIENT_ID environment variable not set")
	}

	auth0ClientSecret := os.Getenv("AUTH0_CLIENT_SECRET")
	if auth0ClientSecret == "" {
		panic("AUTH0_CLIENT_SECRET environment variable not set")
	}

	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	if auth0Domain == "" {
		panic("AUTH0_DOMAIN environment variable not set")
	}

	auth0Audience := os.Getenv("AUTH0_AUDIENCE")
	if auth0Audience == "" {
		panic("AUTH0_AUDIENCE environment variable not set")
	}

	auth0IssuerURL, err := url.Parse("https://" + auth0Domain + "/")
	if err != nil {
		log.Fatalf("Failed to parse Auth0 issuer URL: %v", err)
	}

	I = Config{
		Debug:          os.Getenv("DEBUG") == "1",
		Port:           getPort(),
		Scheduler:      gocron.NewScheduler(time.UTC),
		MaxInviteCount: maxInviteCount,
		Auth0: Auth0Config{
			ClientID:           auth0ClientID,
			ClientSecret:       auth0ClientSecret,
			Domain:             auth0Domain,
			Audience:           auth0Audience,
			IssuerURL:          auth0IssuerURL,
			ManagementAudience: fmt.Sprintf("https://%s/api/v2/", auth0Domain),
		},
	}
}
