package controllers

import (
	"log"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/exp/slices"
)

// Returns basic information about the server, acting
// as a general health endpoint.
func HealthCheck(c *fiber.Ctx) error {
	// Get build info
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		log.Printf("Failed to read build info")
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	depIdx := slices.IndexFunc(bi.Deps, func(d *debug.Module) bool {
		return d.Path == "github.com/joshnies/decent-vcs"
	})

	if depIdx == -1 {
		log.Printf("Failed to determine Go module version for health endpoint")
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"version": bi.Deps[depIdx].Version,
	})
}
