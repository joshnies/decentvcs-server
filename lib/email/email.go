package email

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joshnies/decent-vcs-api/config"
)

type SendEmailOptions struct {
	From    string
	To      string
	Subject string
	Body    string
}

// Send an email.
func Send(options SendEmailOptions) error {
	if config.EmailClient.Mailgun != nil {
		return sendMailgun(options)
	}

	// TODO: Integrate SendGrid

	log.Fatal("No email client initialized")
	return nil
}

func sendMailgun(options SendEmailOptions) error {
	client := config.EmailClient.Mailgun
	msg := client.NewMessage(options.From, options.Subject, options.Body, options.To)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, _, err := client.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("Error sending email: %s", err.Error())
	}

	return nil
}
