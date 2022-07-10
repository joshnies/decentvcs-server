package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/acl"
	"github.com/joshnies/decent-vcs/lib/auth"
	"github.com/joshnies/decent-vcs/models"
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

	if len(reqBody.Paths) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "No paths provided",
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

	// Get user ID
	userID, err := auth.GetUserID(c)
	if err != nil {
		return err
	}

	locks := make(map[string]string)
	if branch.Locks != nil {
		locks = branch.Locks
	}

	for _, path := range reqBody.Paths {
		// Check if file is already locked
		if val, ok := branch.Locks[path]; ok {
			lockedBy := "(unknown)"

			// Get name of user who locked the file
			if val != userID {
				stytchUser, err := auth.GetStytchUserByID(val)
				if err != nil {
					fmt.Printf("Error while getting Stytch user who locked a file: %v\n", err)
				} else {
					lockedBy = stytchUser.Name.FirstName + " " + stytchUser.Name.LastName
				}
			} else {
				lockedBy = "you"
			}

			// Return error
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": fmt.Sprintf("Path \"%s\" is already locked by %s", path, lockedBy),
			})
		}

		// File is available to lock
		locks[path] = userID
	}

	// Add locked paths to branch
	if _, err := config.MI.DB.Collection("branches").UpdateByID(ctx, bid, bson.M{"$set": bson.M{"locks": locks}}); err != nil {
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

	if len(reqBody.Paths) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "No paths provided",
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

	if branch.Locks == nil || len(branch.Locks) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Branch has no locks",
		})
	}

	// Get user ID
	userID, err := auth.GetUserID(c)
	if err != nil {
		return err
	}

	// Get "force" query param
	force := c.Query("force") == "true"

	// Remove locked paths from branch
	for _, path := range reqBody.Paths {
		// Make sure user is the current locker
		if val, ok := branch.Locks[path]; ok {
			if val != userID {
				// User is not the file locker.
				//
				// Return error if:
				// - "force" was not provided
				// - "force" was provided but user is not an admin or higher for the project
				if force {
					canForceUnlock, err := acl.HasProjectAccess(userID, branch.ProjectID.Hex(), models.RoleAdmin)
					if err != nil || !canForceUnlock {
						return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
							"error":   "Forbidden",
							"message": "You do not have permission to force unlock files",
						})
					}
				} else {
					lockedBy := "(unknown)"

					// Get name of user who locked the file
					if val != userID {
						stytchUser, err := auth.GetStytchUserByID(val)
						if err != nil {
							fmt.Printf("Error while getting Stytch user who locked a file: %v\n", err)
						} else {
							lockedBy = stytchUser.Name.FirstName + " " + stytchUser.Name.LastName
						}
					} else {
						lockedBy = "you"
					}

					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
						"error": "Bad request",
						"message": fmt.Sprintf("The file \"%s\" was locked by %s (not you), and cannot be modified until "+
							"the user unlocks it. You may forcefully unlock the file if you are an admin on this project.",
							path,
							lockedBy,
						),
					})
				}
			}

			// Delete file lock from branch (a.k.a. unlock the file)
			delete(branch.Locks, path)
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": fmt.Sprintf("Path \"%s\" is already unlocked", path),
			})
		}
	}

	// Update branch in database
	if _, err := config.MI.DB.Collection("branches").UpdateByID(ctx, bid, bson.M{"$set": bson.M{"locks": branch.Locks}}); err != nil {
		fmt.Printf("Error while updating branch with ID \"%s\": %v\n", bid.Hex(), err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return nil
}
