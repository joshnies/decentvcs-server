package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/joshnies/decent-vcs/config"
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
	config.InitStytch()
	config.InitValidator()

	// Create Fiber instance
	app := fiber.New(fiber.Config{
		AppName: "DecentVCS Server v0.1.0",
	})

	if config.I.Debug {
		app.Use(logger.New())
	}

	// Define routes
	routes.RouteAuth(app.Group("/"))
	routes.RouteProjects(app.Group("/projects", middleware.IsAuthenticated))
	routes.RouteCommits(app.Group("/projects/:pid/commits", middleware.IsAuthenticated, middleware.HasProjectAccess))
	routes.RouteBranches(app.Group("/projects/:pid/branches", middleware.IsAuthenticated, middleware.HasProjectAccess))
	routes.RouteStorage(app.Group("/projects/:pid/storage", middleware.IsAuthenticated, middleware.HasProjectAccess))

	// Start server
	app.Listen(fmt.Sprintf(":%s", config.I.Port))

	// After server stops:
	// Close database connection
	if err := config.MI.Client.Disconnect(ctx); err != nil {
		panic(err)
	}
}
