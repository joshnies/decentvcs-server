package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/decentvcs/server/config"
	"github.com/decentvcs/server/constants"
	"github.com/decentvcs/server/lib/auth"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Create a new access key.
func CreateAccessKey(c *fiber.Ctx) error {
	// Get user & team from context
	userData := auth.GetUserDataFromContext(c)
	team := c.UserContext().Value(models.ContextKeyTeam).(*models.Team)

	// Create access key in database
	accessKeyID := primitive.NewObjectID()
	accessKey := models.AccessKey{
		ID:        accessKeyID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour * 24), // expires in 24 hours after creation
		UserID:    userData.UserID,
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

// Delete the current request's access key.
func DeleteAccessKey(c *fiber.Ctx) error {
	// Get access key ID from header
	accessKeyIDHex := c.Get(constants.AccessKeyHeader)
	if accessKeyIDHex == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(map[string]string{
			"error": "Unauthorized",
		})
	}

	accessKeyID, err := primitive.ObjectIDFromHex(accessKeyIDHex)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(map[string]string{
			"error": "Unauthorized",
		})
	}

	// Delete access key from database
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if _, err := config.MI.DB.Collection("access_keys").DeleteOne(ctx, bson.M{"id": accessKeyID}); err != nil {
		fmt.Printf("[controllers.DeleteAccessKey] Failed to delete access key: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
			"error": "Internal server error",
		})
	}

	return c.JSON(map[string]string{
		"message": "Access key deleted",
	})
}
