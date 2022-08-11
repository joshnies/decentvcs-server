package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/auth"
	"github.com/joshnies/decent-vcs/models"
	"go.mongodb.org/mongo-driver/bson"
)

// Get user data for a single user.
func GetUserData(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(auth.GetUserDataFromContext(c))
}

// Update user data.
func UpdateUserData(c *fiber.Ctx) error {
	userData := auth.GetUserDataFromContext(c)

	// Parse request body
	var reqBody models.UpdateUserDataRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Update user data
	if _, err := config.MI.DB.Collection("user_data").UpdateByID(
		ctx,
		userData.ID,
		bson.M{
			"set": bson.M{
				"avatar_url": reqBody.AvatarURL,
			},
		},
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}

	return nil
}
