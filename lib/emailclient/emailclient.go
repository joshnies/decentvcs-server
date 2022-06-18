package emailclient

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joshnies/decent-vcs/config"
)

type SendEmailOptions struct {
	From         string
	To           []string
	Subject      string
	Body         string
	Template     string
	TemplateVars map[string]any
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
	body := options.Body
	if options.Template != "" {
		body = ""
	}

	msg := client.NewMessage(options.From, options.Subject, body, options.To...)
	if options.Template != "" {
		msg.SetTemplate(options.Template)

		for k, v := range options.TemplateVars {
			msg.AddTemplateVariable(k, v)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, _, err := client.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("error sending email: %s", err.Error())
	}

	return nil
}
