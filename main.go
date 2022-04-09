package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/joshnies/qc-api/config"
	"github.com/joshnies/qc-api/routes/projects"
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

	// Define routes
	// /v1
	v1 := app.Group("/v1")
	// /v1/projects
	v1Projects := v1.Group("/projects")
	v1Projects.Get("/", projects.GetManyProjects)
	v1Projects.Get("/:id", projects.GetOneProject)
	v1Projects.Post("/", projects.CreateProject)

	// Start server
	app.Listen(":8000")

	// Close database connection
	if err := config.MI.Client.Disconnect(context.TODO()); err != nil {
		panic(err)
	}
}
