package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/decentvcs/server/config"
	"github.com/decentvcs/server/lib/acl"
	"github.com/decentvcs/server/lib/auth"
	"github.com/decentvcs/server/lib/branch_lib"
	"github.com/decentvcs/server/lib/team_lib"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Lock one or many files from edits by other users.
func Lock(c *fiber.Ctx) error {
	userData := auth.GetUserDataFromContext(c)
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")
	branchName := c.Params("branch_name")

	// Parse request body
	var reqBody models.LockOrUnlockRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return err
	}

	// Validate request body
	if err := config.Validator.Struct(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if len(reqBody.Paths) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No paths provided",
		})
	}

	// Get branch with commit
	branch, err := branch_lib.GetOneWithCommit(team.ID, projectName, branchName)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Branch not found",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	var filePaths []string
	for _, path := range reqBody.Paths {
		// Make sure path exists in branch remote
		if _, ok := branch.Commit.Files[path]; ok {
			// File path exists
			filePaths = append(filePaths, path)
		} else {
			// File path does not exist in branch remote, check if path is a directory
			found := false
			for _, key := range lo.Keys(branch.Commit.Files) {
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
			if val != userData.UserID {
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
				"error": fmt.Sprintf("Path \"%s\" is already locked by %s", path, lockedBy),
			})
		}

		// File is available to lock
		locks[path] = userData.UserID
	}

	// Add locked paths to branch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := config.MI.DB.Collection("branches").UpdateByID(ctx, branch.ID, bson.M{"$set": bson.M{"locks": locks}}); err != nil {
		fmt.Printf("[Lock] Error updating branch: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return nil
}

// Remove the lock on one or many files, allowing other users on the project to edit them again.
func Unlock(c *fiber.Ctx) error {
	userData := auth.GetUserDataFromContext(c)
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")
	branchName := c.Params("branch_name")

	// Parse request body
	var reqBody models.LockOrUnlockRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return err
	}

	// Validate request body
	if err := config.Validator.Struct(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if len(reqBody.Paths) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No paths provided",
		})
	}

	// Get branch with commit
	branch, err := branch_lib.GetOneWithCommit(team.ID, projectName, branchName)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Branch not found",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	if branch.Locks == nil || len(branch.Locks) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Branch has no locks",
		})
	}

	var filePaths []string
	for _, path := range reqBody.Paths {
		// Make sure path exists in branch remote
		if _, ok := branch.Commit.Files[path]; ok {
			// File path exists
			filePaths = append(filePaths, path)
		} else {
			// File path does not exist in branch remote, check if path is a directory
			found := false
			for _, key := range lo.Keys(branch.Commit.Files) {
				// Path is a directory, add all committed files in directory
				if strings.HasPrefix(key, path) {
					filePaths = append(filePaths, key)
					found = true
				}
			}

			if !found {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("File \"%s\" is not a file or directory in remote branch \"%s\"", path, branch.Name),
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
			if val != userData.UserID {
				// User is not the file locker.
				//
				// Return error if:
				// - "force" was not provided
				// - "force" was provided but user is not an admin or higher for the project
				if force {
					teamAccess, err := acl.HasTeamAccess(userData, team.Name, models.RoleAdmin)
					if err != nil || !teamAccess.HasAccess {
						return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
							"error": "You do not have permission to force unlock files",
						})
					}
				} else {
					lockedBy := "(unknown)"

					// Get name of user who locked the file
					if val != userData.UserID {
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
				"error": fmt.Sprintf("Path \"%s\" is already unlocked", path),
			})
		}
	}

	// Update branch in database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := config.MI.DB.Collection("branches").UpdateByID(ctx, branch.ID, bson.M{"$set": bson.M{"locks": branch.Locks}}); err != nil {
		fmt.Printf("[Unlock] Error updating branch: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return nil
}
