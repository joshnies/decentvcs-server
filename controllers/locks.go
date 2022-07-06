package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/models"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Lock one or many files from edits by other users.
//
// URL params:
//
// - pid: project ID
//
// - bid: branch ID
//
func Lock(c *fiber.Ctx) error {
	// Get branch ID
	bidStr := c.Params("bid")
	bid, err := primitive.ObjectIDFromHex(bidStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid branch ID",
		})
	}

	// Parse request body
	var reqBody models.LockOrUnlockRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return err
	}

	// Validate request body
	if err := config.Validator.Struct(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": err.Error(),
		})
	}

	// Get branch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var branch models.Branch
	if err := config.MI.DB.Collection("branches").FindOne(ctx, bson.M{"_id": bid}).Decode(&branch); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Not found",
			"message": "Branch not found",
		})
	}

	// Add locked paths to branch
	// NOTE: Silently ignores paths that are already locked
	newLockedPaths := lo.Uniq(append(branch.LockedPaths, reqBody.Paths...))
	if _, err := config.MI.DB.Collection("branches").UpdateByID(ctx, bid, &models.Branch{LockedPaths: newLockedPaths}); err != nil {
		fmt.Printf("Error while updating branch with ID \"%s\": %v\n", bid.Hex(), err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return nil
}

// Remove the lock on one or many files, allowing other users on the project to edit them again.
func Unlock(c *fiber.Ctx) error {
	// Get branch ID
	bidStr := c.Params("bid")
	bid, err := primitive.ObjectIDFromHex(bidStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid branch ID",
		})
	}

	// Parse request body
	var reqBody models.LockOrUnlockRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return err
	}

	// Validate request body
	if err := config.Validator.Struct(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": err.Error(),
		})
	}

	// Get branch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var branch models.Branch
	if err := config.MI.DB.Collection("branches").FindOne(ctx, bson.M{"_id": bid}).Decode(&branch); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Not found",
			"message": "Branch not found",
		})
	}

	// Remove locked paths from branch
	// NOTE: Silently ignores paths that are not locked
	newLockedPaths := lo.Filter(branch.LockedPaths, func(pathToRm string, _ int) bool {
		return !lo.Contains(reqBody.Paths, pathToRm)
	})

	if _, err := config.MI.DB.Collection("branches").UpdateByID(ctx, bid, &models.Branch{LockedPaths: newLockedPaths}); err != nil {
		fmt.Printf("Error while updating branch with ID \"%s\": %v\n", bid.Hex(), err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return nil
}
