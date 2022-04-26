package config

import "os"

type Config struct {
	Verbose bool
	Port    string
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
	I = Config{
		Verbose: os.Getenv("VERBOSE") == "true",
		Port:    getPort(),
	}
}
