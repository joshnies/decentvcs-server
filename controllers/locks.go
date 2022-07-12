package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/acl"
	"github.com/joshnies/decent-vcs/lib/auth"
	"github.com/joshnies/decent-vcs/lib/branch_lib"
	"github.com/joshnies/decent-vcs/models"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
	// Get project ID
	pidStr := c.Params("pid")
	pid, err := primitive.ObjectIDFromHex(pidStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

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

	// Get branch with commit
	branch, err := branch_lib.GetOneWithCommit(pid, bid)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "Not found",
				"message": "Branch not found",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get user ID
	userID, err := auth.GetUserID(c)
	if err != nil {
		return err
	}

	var filePaths []string
	for _, path := range reqBody.Paths {
		// Make sure path exists in branch remote
		if _, ok := branch.Commit.HashMap[path]; ok {
			// File path exists
			filePaths = append(filePaths, path)
		} else {
			// File path does not exist in branch remote, check if path is a directory
			found := false
			for _, key := range lo.Keys(branch.Commit.HashMap) {
				// Path is a directory, add all committed files in directory
				if strings.HasPrefix(key, path) {
					filePaths = append(filePaths, key)
					found = true
				}
			}

			if !found {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "Bad request",
					"message": fmt.Sprintf("File \"%s\" is not a file or directory in remote branch \"%s\"", path, branch.Name),
				})
			}
		}
	}

	locks := make(map[string]string)
	if branch.Locks != nil {
		locks = branch.Locks
	}

	for _, path := range filePaths {
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
	// Get project ID
	pidStr := c.Params("pid")
	pid, err := primitive.ObjectIDFromHex(pidStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

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

	// Get branch with commit
	branch, err := branch_lib.GetOneWithCommit(pid, bid)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "Not found",
				"message": "Branch not found",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
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

	var filePaths []string
	for _, path := range reqBody.Paths {
		// Make sure path exists in branch remote
		if _, ok := branch.Commit.HashMap[path]; ok {
			// File path exists
			filePaths = append(filePaths, path)
		} else {
			// File path does not exist in branch remote, check if path is a directory
			found := false
			for _, key := range lo.Keys(branch.Commit.HashMap) {
				// Path is a directory, add all committed files in directory
				if strings.HasPrefix(key, path) {
					filePaths = append(filePaths, key)
					found = true
				}
			}

			if !found {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "Bad request",
					"message": fmt.Sprintf("File \"%s\" is not a file or directory in remote branch \"%s\"", path, branch.Name),
				})
			}
		}
	}

	// Get "force" query param
	force := c.Query("force") == "true"

	// Remove locked paths from branch
	for _, path := range filePaths {
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := config.MI.DB.Collection("branches").UpdateByID(ctx, bid, bson.M{"$set": bson.M{"locks": branch.Locks}}); err != nil {
		fmt.Printf("Error while updating branch with ID \"%s\": %v\n", bid.Hex(), err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return nil
}
