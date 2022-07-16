package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/joshnies/decent-vcs/config"
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
	routes.RouteStytch(app.Group("/stytch"))
	routes.RouteTeams(app.Group("/teams"))
	routes.RouteProjects(app.Group("projects/:team_name/projects"))

	projectGroup := app.Group("/projects/:team_name/:project_name")
	routes.RouteBranches(projectGroup.Group("/branches"))
	routes.RouteCommits(projectGroup.Group("/commits"))
	routes.RouteStorage(projectGroup.Group("/storage"))

	// Start server
	app.Listen(fmt.Sprintf(":%s", config.I.Port))

	// After server stops:
	// Close database connection
	if err := config.MI.Client.Disconnect(ctx); err != nil {
		panic(err)
	}
}
