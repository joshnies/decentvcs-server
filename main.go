package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/auth"
	"github.com/joshnies/decent-vcs/middleware"
	"github.com/joshnies/decent-vcs/routes"
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
	config.InitEmailClient()

	// Fetch initial Auth0 management API access token
	auth.UpdateAuth0ManagementToken()

	// Create Fiber instance
	app := fiber.New(fiber.Config{
		AppName: "DecentVCS Server v0.1.0",
	})

	// Use middleware
	app.Use(adaptor.HTTPMiddleware(middleware.ValidateJWT()))

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
