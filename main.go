package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/joshnies/qc-api/config"
	"github.com/joshnies/qc-api/routes"
)

func main() {
	// Load environment variables
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize database client
	config.InitDatabase()

	// Create Fiber instance
	app := fiber.New(fiber.Config{
		AppName: "Quanta Control API v0.1.0",
	})

	// Use middleware
	app.Use(logger.New())

	// Define v1 routes
	v1 := app.Group("/v1")
	routes.RouteProjects(v1.Group("/projects"))
	routes.RouteBranches(v1.Group("/branches"))
	routes.RouteCommits(v1.Group("/commits"))

	// Start server
	app.Listen(fmt.Sprintf(":%s", config.GetPort()))

	// After server stops:
	// Close database connection
	if err := config.MI.Client.Disconnect(context.TODO()); err != nil {
		panic(err)
	}
}
