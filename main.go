package main

import (
	"context"
	"fmt"
	"time"

	"github.com/decentvcs/server/config"
	"github.com/decentvcs/server/constants"
	"github.com/decentvcs/server/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
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
		AppName: "DecentVCS Server v1.0.0",
	})

	// Configure global middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: fmt.Sprintf("Origin, Content-Type, Accept, %s", constants.SessionTokenHeader),
	}))

	if config.I.Debug {
		app.Use(logger.New())
	}

	// Define routes
	routes.RouteRoot(app.Group("/"))
	routes.RouteStytch(app.Group("/stytch"))
	routes.RouteUserData(app.Group("/users"))
	routes.RouteTeams(app.Group("/teams"))
	routes.RouteAccessKeys(app.Group("/teams/:team_name/access_keys"))

	projectGroup := app.Group("/projects/:team_name/:project_name")
	routes.RouteProjects(projectGroup)
	routes.RouteBranches(projectGroup.Group("/branches"))
	routes.RouteCommits(projectGroup.Group("/commits"))
	routes.RouteStorage(projectGroup.Group("/storage"))

	// Start server
	app.Listen(fmt.Sprintf(":%d", config.I.Port))

	// After server stops:
	// Close database connection
	if err := config.MI.Client.Disconnect(ctx); err != nil {
		panic(err)
	}
}
