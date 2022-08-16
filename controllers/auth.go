package controllers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/constants"
	"github.com/joshnies/decent-vcs/models"
	"github.com/stytchauth/stytch-go/v5/stytch"
)

// Create or refresh a Stytch session.
// This is usually only needed to refresh an existing session, since the website handles all auth flows.
//
// To refresh an existing session, provide `SessionToken`.
func CreateOrRefreshSession(c *fiber.Ctx) error {
	// Validate request body
	var body models.AuthenticateRequest
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	var sessionToken string
	if body.TokenType == "magic_links" {
		// Authenticate magic link token
		stytchres, err := config.StytchClient.MagicLinks.Authenticate(&stytch.MagicLinksAuthenticateParams{
			Token:                  body.Token,
			SessionToken:           body.SessionToken,
			SessionDurationMinutes: config.I.Stytch.SessionDurationMinutes,
			Attributes: stytch.Attributes{
				IPAddress: c.IP(),
			},
		})
		if err != nil {
			return err
		}

		sessionToken = stytchres.SessionToken
	} else if body.TokenType == "oauth" {
		// Authenticate OAuth token
		stytchres, err := config.StytchClient.OAuth.Authenticate(&stytch.OAuthAuthenticateParams{
			Token:                  body.Token,
			SessionToken:           body.SessionToken,
			SessionDurationMinutes: config.I.Stytch.SessionDurationMinutes,
		})
		if err != nil {
			return err
		}

		sessionToken = stytchres.SessionToken
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid token type \"%s\"; must be either `magic_link` or `oauth`", body.TokenType),
		})
	}

	// Return response
	res := models.AuthenticateResponse{
		SessionToken: sessionToken,
	}

	return c.JSON(res)
}

// Revoke the session.
func RevokeSession(c *fiber.Ctx) error {
	sessionToken := c.Get(constants.SessionTokenHeader) // already validated in the `ValidateAuth` middleware

	_, err := config.StytchClient.Sessions.Revoke(&stytch.SessionsRevokeParams{
		SessionToken: sessionToken,
	})

	return err
}
