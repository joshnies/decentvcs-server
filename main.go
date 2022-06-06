package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/joshnies/decent-vcs-api/config"
	"github.com/joshnies/decent-vcs-api/middleware"
	"github.com/joshnies/decent-vcs-api/routes"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Load environment variables
	// NOTE: We're ignoring errors here because we don't care if the .env file doesn't exist.
	// Decent uses a "bring your own environment" approach.
	godotenv.Load()

	// Initialize stuff
	config.InitConfig()
	config.InitDatabase()
	config.InitStorage()

	// Create Fiber instance
	app := fiber.New(fiber.Config{
		AppName: "DecentVCS API v0.1.0",
	})

	// Use middleware
	app.Use(adaptor.HTTPMiddleware(middleware.EnsureValidToken()))

	if config.I.Debug {
		app.Use(logger.New())
	}

	// Define routes
	routes.RouteProjects(app.Group("/projects"))
	routes.RouteCommits(app.Group("/projects/:pid/commits"))
	routes.RouteBranches(app.Group("/projects/:pid/branches"))
	routes.RouteStorage(app.Group("/projects/:pid/storage"))

	// Start server
	app.Listen(fmt.Sprintf(":%s", config.I.Port))

	// After server stops:
	// Close database connection
	if err := config.MI.Client.Disconnect(ctx); err != nil {
		panic(err)
	}
}
