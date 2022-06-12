package controllers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/mail"
	"time"

	"github.com/go-jose/go-jose/json"
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs-api/config"
	"github.com/joshnies/decent-vcs-api/lib/acl"
	"github.com/joshnies/decent-vcs-api/lib/auth"
	"github.com/joshnies/decent-vcs-api/models"
	"github.com/joshnies/decent-vcs-api/models/auth0"
	"github.com/sethvargo/go-password/password"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Get one project by ID.
//
// URL params:
//
// - id: Project ID
//
func GetOneProject(c *fiber.Ctx) error {
	// Get project ID
	pid := c.Params("pid")
	projectObjId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get project from database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result models.Project
	err = config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"_id": projectObjId}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Not found",
		})
	}
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(result)
}

// Get one project by blob.
//
// URL params:
//
// - oa: Alias of the user or team who owns the project
//
// - pname: Name of the project
//
func GetOneProjectByBlob(c *fiber.Ctx) error {
	// ownerAlias := c.Params("oa")
	projectName := c.Params("pname")

	// Get project from database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result models.Project
	err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"name": projectName}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Not found",
		})
	}
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Check if user has access to project
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	hasAccess, err := acl.HasProjectAccess(userId, result.ID.Hex())
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	if !hasAccess {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	return c.JSON(result)
}

// Create a new project.
//
// Body:
//
// - name: Project name
//
func CreateProject(c *fiber.Ctx) error {
	// Get user ID
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse body
	var body models.Project
	if err := c.BodyParser(&body); err != nil {
		fmt.Println(err) // DEBUG
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Validate body
	if vErr := validate.Struct(body); vErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": vErr.Error(),
		})
	}

	// Generate default branch ID ahead of time
	branchId := primitive.NewObjectID()

	// Create new project
	project := models.Project{
		ID:              primitive.NewObjectID(),
		CreatedAt:       time.Now().Unix(),
		OwnerID:         userID,
		Name:            body.Name,
		DefaultBranchID: branchId,
	}

	// Create project in database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = config.MI.DB.Collection("projects").InsertOne(ctx, project)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create initial commit
	commit := models.Commit{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().Unix(),
		Index:     1, // Starts at 1 since in Go, 0 is the default and used to check for empty values
		ProjectID: project.ID,
		BranchID:  branchId,
		Message:   "Initial commit",
	}

	_, err = config.MI.DB.Collection("commits").InsertOne(ctx, commit)
	if err != nil {
		// Delete project
		config.MI.DB.Collection("projects").DeleteOne(ctx, bson.M{"_id": project.ID})

		// Output error
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create default branch
	branch := models.BranchCreateBSON{
		ID:        branchId,
		CreatedAt: time.Now().Unix(),
		Name:      "stable",
		ProjectID: project.ID,
		CommitID:  commit.ID,
	}

	// Insert branch into database
	_, err = config.MI.DB.Collection("branches").InsertOne(ctx, branch)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Add `owner` permission to user `app_metadata` in Auth0
	httpClient := &http.Client{}
	reqBody, _ := json.Marshal(map[string]any{
		"app_metadata": map[string]any{
			fmt.Sprintf("permission:%s:owner", project.ID.Hex()): true,
		},
	})
	req, _ := http.NewRequest(
		"PATCH",
		fmt.Sprintf("https://%s/api/v2/users/%s", config.I.Auth0.Domain, userID),
		bytes.NewBuffer(reqBody),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth0.ManagementToken))
	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	if res.StatusCode != 200 {
		fmt.Printf("Error status code recieved from Auth0 while adding user permission: %d\n", res.StatusCode)

		// Dump response
		dump, _ := httputil.DumpResponse(res, true)
		fmt.Println(string(dump))

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"_id":  project.ID.Hex(),
		"name": project.Name,
		"branches": []fiber.Map{
			{
				"_id":  branch.ID.Hex(),
				"name": branch.Name,
				"commit": fiber.Map{
					"_id":   commit.ID.Hex(),
					"index": commit.Index,
				},
			},
		},
	})
}

// Update a project.
//
// URL params:
//
// - pid: Project ID
//
// Body: (any field from Project)
//
// Returns the updated project.
func UpdateOneProject(c *fiber.Ctx) error {
	// Parse project ID
	pid := c.Params("pid")
	projectObjectId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Make sure user is a project admin
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	role, err := acl.GetProjectRole(userId, pid)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	if role != acl.RoleAdmin {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse body
	var body models.Project
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	updateData := bson.M{}
	if body.OwnerID != "" {
		updateData["owner_id"] = body.OwnerID
	}
	if body.Name != "" {
		updateData["name"] = body.Name
	}
	if body.DefaultBranchID != primitive.NilObjectID {
		updateData["default_branch_id"] = body.DefaultBranchID
	}

	// Update project
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = config.MI.DB.Collection("projects").UpdateOne(
		ctx,
		bson.M{"_id": projectObjectId},
		bson.M{"$set": updateData},
	)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"_id":               projectObjectId.Hex(),
		"owner_id":          body.OwnerID,
		"name":              body.Name,
		"default_branch_id": body.DefaultBranchID.Hex(),
	})
}

// Delete project and all of its subresources.
func DeleteOneProject(c *fiber.Ctx) error {
	// Get project
	pid := c.Params("pid")
	projectObjectId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get project
	project := models.Project{}
	err = config.MI.DB.Collection("projects").FindOne(context.Background(), bson.M{"_id": projectObjectId}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Project not found",
		})
	}
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete all commits for project
	_, err = config.MI.DB.Collection("commits").DeleteMany(context.Background(), bson.M{"project_id": project.ID})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete all branches for project
	_, err = config.MI.DB.Collection("branches").DeleteMany(context.Background(), bson.M{"project_id": project.ID})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete project
	_, err = config.MI.DB.Collection("projects").DeleteOne(context.Background(), bson.M{"_id": projectObjectId})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Project deleted",
	})
}

// Invite many users to a project.
func InviteManyUsers(c *fiber.Ctx) error {
	// Validate project
	pid := c.Params("pid")
	_, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Parse request body
	var body models.InviteManyUsersDTO
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Limit email count to prevent request timeouts
	if len(body.Emails) > config.I.MaxInviteCount {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": fmt.Sprintf("Too many emails; limit: %d", config.I.MaxInviteCount),
		})
	}

	// Validate emails in request body
	for _, email := range body.Emails {
		if _, err := mail.ParseAddress(email); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad request",
				"message": fmt.Sprintf("Invalid email: %s", email),
			})
		}
	}

	// Invite new users and add permission for existing users
	httpClient := &http.Client{}
	for _, email := range body.Emails {
		req, _ := http.NewRequest(
			"GET",
			fmt.Sprintf("https://%s/api/v2/users?search_engine=v3&include_fields=true&q=email:%s", config.I.Auth0.Domain, email),
			nil,
		)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth0.ManagementToken))

		res, err := httpClient.Do(req)
		if err != nil {
			fmt.Println(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
		if res.StatusCode != 200 {
			fmt.Printf("Received status code %d from Auth0 while searching for user with email \"%s\"\n", res.StatusCode, email)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		var users []map[string]any
		err = json.NewDecoder(res.Body).Decode(&users)
		if err != nil {
			fmt.Println(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		if len(users) == 0 {
			// User doesn't exist in system yet, invite them
			pwd, err := password.Generate(32, 5, 5, false, false)
			if err != nil {
				fmt.Println(err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			body, _ := json.Marshal(map[string]any{
				"connection":   "Username-Password-Authentication",
				"email":        email,
				"password":     pwd,   // use random password, user will need to reset via invite email
				"verify_email": false, // email will be verified via the password change ticket
				"app_metadata": map[string]any{
					fmt.Sprintf("permission:%s:collab", pid): true,
				},
			})
			req, _ := http.NewRequest(
				"POST",
				fmt.Sprintf("https://%s/api/v2/users", config.I.Auth0.Domain),
				bytes.NewBuffer(body),
			)
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth0.ManagementToken))
			req.Header.Add("Content-Type", "application/json")
			res, err := httpClient.Do(req)
			if err != nil {
				fmt.Println(err)

				// Dump response
				dump, _ := httputil.DumpResponse(res, true)
				fmt.Println(string(dump))

				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
			if res.StatusCode != 201 {
				fmt.Printf("Received status code %d from Auth0 while inviting user with email \"%s\"\n", res.StatusCode, email)

				// Dump response
				dump, _ := httputil.DumpResponse(res, true)
				fmt.Println(string(dump))

				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
			defer res.Body.Close()

			// Parse body
			var resBody auth0.CreateUserResponse
			err = json.NewDecoder(res.Body).Decode(&resBody)
			if err != nil {
				fmt.Println(err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// Create password change ticket in Auth0
			identity := resBody.Identities[0]
			body, _ = json.Marshal(map[string]any{
				"user_id":                fmt.Sprintf("%s|%s", identity.Provider, identity.UserID),
				"mark_email_as_verified": true,
				"return_url":             config.I.Auth0.InviteReturnURL,
			})
			req, _ = http.NewRequest(
				"POST",
				fmt.Sprintf("https://%s/api/v2/tickets/password-change", config.I.Auth0.Domain),
				bytes.NewBuffer(body),
			)
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth0.ManagementToken))
			req.Header.Add("Content-Type", "application/json")
			res, err = httpClient.Do(req)
			if err != nil {
				fmt.Println(err)

				// Dump response
				dump, _ := httputil.DumpResponse(res, true)
				fmt.Println(string(dump))

				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
			if res.StatusCode != 201 {
				fmt.Printf("Received status code %d from Auth0 while creating password change ticket for user with email \"%s\"\n", res.StatusCode, email)

				// Dump response
				dump, _ := httputil.DumpResponse(res, true)
				fmt.Println(string(dump))

				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// Parse body
			var ticketRes auth0.CreatePasswordChangeTicketResponse
			err = json.NewDecoder(res.Body).Decode(&ticketRes)
			if err != nil {
				fmt.Println(err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// TODO: Send invite email
		} else {
			// Add permission for existing user
			user := users[0]
			body, _ := json.Marshal(map[string]any{
				"app_metadata": map[string]any{
					fmt.Sprintf("permission:%s:collab", pid): true,
				},
			})
			req, _ := http.NewRequest(
				"PATCH",
				fmt.Sprintf("https://%s/api/v2/users/%s", config.I.Auth0.Domain, user["user_id"]),
				bytes.NewBuffer(body),
			)
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth0.ManagementToken))
			req.Header.Add("Content-Type", "application/json")
			res, err := httpClient.Do(req)
			if err != nil {
				fmt.Println(err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
			if res.StatusCode != 200 {
				fmt.Printf("Received status code %d from Auth0 while adding permission for user with email \"%s\"\n", res.StatusCode, email)

				// Dump response
				dump, _ := httputil.DumpResponse(res, true)
				fmt.Println(string(dump))

				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			// TODO: Send existing user an email notifying them of their new permission
		}
	}

	return nil
}
