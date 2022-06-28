package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/models"
	"github.com/stytchauth/stytch-go/v5/stytch"
)

// Authenticate Stytch session token.
func Authenticate(c *fiber.Ctx) error {
	// Validate request body
	var body models.AuthenticateRequest
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	// Authenticate magic link token
	//
	// If `SessionToken` is provided, the existing session will be refreshed instead of creating a new one
	stytchres, err := config.StytchClient.MagicLinks.Authenticate(&stytch.MagicLinksAuthenticateParams{
		Token:                  body.Token,
		SessionToken:           body.SessionToken,
		SessionDurationMinutes: config.I.Stytch.SessionDurationMinutes,
		// Options:    stytch.Options{IPMatchRequired: true},
		// Attributes: stytch.Attributes{IPAddress: "10.0.0.0"},
	})
	if err != nil {
		return err
	}

	// Return response
	res := models.AuthenticateResponse{
		SessionToken: stytchres.SessionToken,
	}

	c.Status(200).JSON(res)
	return nil
}
