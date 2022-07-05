package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-co-op/gocron"
)

type EmailTemplatesConfig struct {
	InviteExistingUser string
}

type EmailConfig struct {
	SendGridAPIKey string
	NoReplyEmail   string
	Templates      EmailTemplatesConfig
}

type StytchConfig struct {
	SessionDurationMinutes  int32
	InviteExpirationMinutes int32
	InviteRedirectURL       string
}

type Config struct {
	Debug           bool
	LogResponseBody bool
	Port            string
	Scheduler       *gocron.Scheduler
	MaxInviteCount  int
	Stytch          StytchConfig
	Email           EmailConfig
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

	// Stytch
	sessionDurationMinutesStr := os.Getenv("SESSION_DURATION_MINUTES")
	if sessionDurationMinutesStr == "" {
		sessionDurationMinutesStr = "1440" // 24 hours
	}
	sessionDurationMinutes, err := strconv.Atoi(sessionDurationMinutesStr)
	if err != nil {
		log.Fatal("SESSION_DURATION_MINUTES must be an integer")
	}

	inviteExpStr := os.Getenv("INVITE_EXPIRATION_MINUTES")
	if inviteExpStr == "" {
		inviteExpStr = "1440" // 24 hours
	}
	inviteExp, err := strconv.Atoi(inviteExpStr)
	if err != nil {
		log.Fatal("SESSION_DURATION_MINUTES must be an integer")
	}

	inviteRedirectURL := os.Getenv("INVITE_REDIRECT_URL")

	// SendGrid
	sgApiKey := os.Getenv("SENDGRID_API_KEY")
	if sgApiKey == "" {
		log.Fatal("SENDGRID_API_KEY environment variable is not set")
	}

	I = Config{
		Debug:           os.Getenv("DEBUG") == "1",
		LogResponseBody: os.Getenv("DEBUG_RES") == "1",
		Port:            getPort(),
		Scheduler:       gocron.NewScheduler(time.UTC),
		MaxInviteCount:  maxInviteCount,
		Stytch: StytchConfig{
			SessionDurationMinutes:  int32(sessionDurationMinutes),
			InviteExpirationMinutes: int32(inviteExp),
			InviteRedirectURL:       inviteRedirectURL,
		},
		Email: EmailConfig{
			SendGridAPIKey: sgApiKey,
			NoReplyEmail:   "no-reply@decentvcs.com",
			Templates: EmailTemplatesConfig{
				InviteExistingUser: "d-14be2f90a89745fbb53d531e80fd9a14",
			},
		},
	}
}
