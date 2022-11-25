package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/stripe/stripe-go/v74"
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

type StripeConfig struct {
	CloudPlanPriceID string
}

type Config struct {
	Debug           bool
	LogResponseBody bool
	Port            uint
	Scheduler       *gocron.Scheduler
	MaxInviteCount  int
	Stytch          StytchConfig
	Email           EmailConfig
	Stripe          StripeConfig
}

// Global config instance
var I Config

func getPort() uint {
	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8080"
	}

	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		log.Fatal("Invalid PORT environment variable")
	}

	return uint(port)
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

	// Stripe
	stripeApiKey := os.Getenv("STRIPE_API_KEY")
	if stripeApiKey == "" {
		log.Fatal("STRIPE_API_KEY environment variable is not set")
	}

	stripeCloudPlanPriceID := os.Getenv("STRIPE_CLOUD_PLAN_PRICE_ID")
	if stripeCloudPlanPriceID == "" {
		log.Fatal("STRIPE_CLOUD_PLAN_PRICE_ID environment variable is not set")
	}

	// Configure global Stripe instance
	stripe.Key = stripeApiKey

	// Construct and assign config instance
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
		Stripe: StripeConfig{
			CloudPlanPriceID: stripeCloudPlanPriceID,
		},
	}
}
