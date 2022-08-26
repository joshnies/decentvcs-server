package controllers

import (
	"github.com/gofiber/fiber/v2"
)

// Basic ping-pong endpoint that acts as a
// health check for the server.
func Ping(c *fiber.Ctx) error {
	return c.Status(200).JSON(fiber.Map{
		"message": "pong",
	})
}
