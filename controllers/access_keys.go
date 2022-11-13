package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/decentvcs/server/config"
	"github.com/decentvcs/server/constants"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Create a new access key.
func CreateAccessKey(c *fiber.Ctx) error {
	// Get user & team from context
	user := c.UserContext().Value(models.ContextKeyUserData).(*models.UserData)
	team := c.UserContext().Value(models.ContextKeyTeam).(*models.Team)

	// Create access key in database
	accessKeyID := primitive.NewObjectID()
	accessKey := models.AccessKey{
		ID:        accessKeyID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour * 24), // expires in 24 hours after creation
		UserID:    user.UserID,
		TeamID:    team.ID,
		Scope:     constants.ScopeTeamUpdateUsage,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if _, err := config.MI.DB.Collection("access_keys").InsertOne(ctx, accessKey); err != nil {
		fmt.Printf("[controllers.CreateAccessKey] Failed to insert access key: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
			"error": "Internal server error",
		})
	}

	return c.JSON(accessKey)
}
