package config

import (
	"log"
	"os"

	"github.com/mailgun/mailgun-go/v4"
)

type EmailFromAddresses struct {
	NoReply string
}

type EmailClientInstance struct {
	FromAddresses EmailFromAddresses
	Mailgun       *mailgun.MailgunImpl
	// TODO: Integrate SendGrid
}

var EmailClient EmailClientInstance

// Initialize email client instance
func InitEmailClient() {
	// Get and validate environment variables
	mgDomain := os.Getenv("MAILGUN_DOMAIN")
	if mgDomain == "" {
		log.Fatal("MAILGUN_DOMAIN environment variable is not set")
	}

	mgApiKey := os.Getenv("MAILGUN_API_KEY")
	if mgApiKey == "" {
		log.Fatal("MAILGUN_API_KEY environment variable is not set")
	}

	// Build and return instance
	EmailClient = EmailClientInstance{
		FromAddresses: EmailFromAddresses{
			NoReply: "no-reply@" + mgDomain,
		},
		Mailgun: mailgun.NewMailgun(mgDomain, mgApiKey),
	}
}
