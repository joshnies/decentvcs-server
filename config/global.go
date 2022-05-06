package config

import "os"

type Auth0Config struct {
	Domain   string
	Audience string
}

type Config struct {
	Debug bool
	Port  string
	Auth0 Auth0Config
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
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	if auth0Domain == "" {
		panic("AUTH0_DOMAIN environment variable not set")
	}

	auth0Audience := os.Getenv("AUTH0_AUDIENCE")
	if auth0Audience == "" {
		panic("AUTH0_AUDIENCE environment variable not set")
	}

	I = Config{
		Debug: os.Getenv("DEBUG") == "1",
		Port:  getPort(),
		Auth0: Auth0Config{
			Domain:   auth0Domain,
			Audience: auth0Audience,
		},
	}
}
