package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	awstypes "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/decentvcs/server/config"
	"github.com/decentvcs/server/lib/storage"
	"github.com/decentvcs/server/lib/team_lib"
	"github.com/decentvcs/server/models"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Returns an object key for storage provider.
func FormatStorageKey(team models.Team, projectName string, key string) string {
	return fmt.Sprintf("%s/%s/%s", team.Name, projectName, key)
}

// Generate presigned URLs for fetching or uploading multiple objects from/to storage, respectively.
//
// Returns a map of key to PresignResponse.
func PresignMany(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Parse request body
	var body []models.PresignOneRequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Generate presigned URLs
	client := s3.NewPresignClient(config.SI.Client)
	keyUrlMap := make(map[string]models.PresignResponse)

	// TODO: Presign in parallel
	for _, opt := range body {
		remoteKey := FormatStorageKey(*team, projectName, opt.Key)
		method := storage.ToPresignMethod(opt.Method)
		res, err := storage.Presign(ctx, storage.PresignOptions{
			S3PresignClient: client,
			Method:          method,
			Bucket:          config.SI.ProjectsBucket,
			Key:             remoteKey,
			ContentType:     opt.ContentType,
			Multipart:       method == storage.PresignMethodPUT,
			Size:            opt.Size,
			Team:            team,
		})
		if err != nil {
			fmt.Printf("[PresignOne] Error presigning URL: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		keyUrlMap[opt.Key] = res
	}

	return c.JSON(keyUrlMap)
}

// Generate presigned URL(s) for the specified storage object, scoped to a project.
// These URLs are used by the client to upload files to storage without the need for
// access keys or ACL.
//
// Returns an array of presigned URLs.
func PresignOne(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Validate presign method
	method := strings.ToUpper(c.Params("method"))
	if method != "PUT" && method != "GET" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid presign method; must be PUT or GET",
		})
	}

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	project := models.Project{}
	if err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": team.ID, "name": projectName}).Decode(&project); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Project not found",
			})
		}

		fmt.Printf("[PresignOne] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Parse request body
	var body models.PresignOneRequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	client := s3.NewPresignClient(config.SI.Client)
	remoteKey := FormatStorageKey(*team, project.Name, body.Key)

	if method == "PUT" {
		res, err := storage.Presign(ctx, storage.PresignOptions{
			S3PresignClient: client,
			Method:          storage.PresignMethodPUT,
			Bucket:          config.SI.ProjectsBucket,
			Key:             remoteKey,
			ContentType:     body.ContentType,
			Multipart:       true,
			Size:            body.Size,
		})
		if err != nil {
			fmt.Printf("[PresignOne] Error presigning GET URL: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		return c.JSON(res)
	} else if method == "GET" {
		res, err := storage.Presign(ctx, storage.PresignOptions{
			S3PresignClient: client,
			Method:          storage.PresignMethodGET,
			Bucket:          config.SI.ProjectsBucket,
			Key:             remoteKey,
			ContentType:     body.ContentType,
			Team:            team,
		})
		if err != nil {
			fmt.Printf("[PresignOne] Error presigning GET URL: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		return c.JSON(res)
	}

	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "Invalid presign method; must be PUT or GET",
	})
}

// Complete an S3 multipart upload.
// Multipart uploads can be started by generating presigned URLs.
func CompleteMultipartUpload(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	var project models.Project
	if err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": team.ID, "name": projectName}).Decode(&project); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Project not found",
			})
		}

		fmt.Printf("[CompleteMultipartUpload] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Parse request body
	var body models.CompleteMultipartUploadRequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Construct parts
	parts := []awstypes.CompletedPart{}
	for _, part := range body.Parts {
		etag := part.ETag
		parts = append(parts, awstypes.CompletedPart{
			ETag:       &etag,
			PartNumber: part.PartNumber,
		})
	}

	// Complete multipart upload
	key := FormatStorageKey(*team, project.Name, body.Key)
	if _, err := config.SI.Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   &config.SI.ProjectsBucket,
		Key:      &key,
		UploadId: &body.UploadId,
		MultipartUpload: &awstypes.CompletedMultipartUpload{
			Parts: parts,
		},
	}); err != nil {
		fmt.Printf("[CompleteMultipartUpload] Error completing multipart upload: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to complete multipart upload, please make sure the upload ID and parts are correct.",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Success",
	})
}

// Abort an S3 multipart upload.
// Multipart uploads can be started by generating presigned URLs.
func AbortMultipartUpload(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	var project models.Project
	if err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": team.ID, "name": projectName}).Decode(&project); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Project not found",
			})
		}

		fmt.Printf("[AbortMultipartUpload] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Parse request body
	var body models.AbortMultipartUploadRequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Abort multipart upload
	if _, err := config.SI.Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   &config.SI.ProjectsBucket,
		Key:      &body.Key,
		UploadId: &body.UploadId,
	}); err != nil {
		fmt.Printf("[AbortMultipartUpload] Error aborting multipart upload: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to abort multipart upload, please make sure the upload ID is correct.",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Success",
	})
}

// Delete all unused objects from storage based on commit hash maps.
func DeleteUnusedStorageObjects(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	var project models.Project
	if err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": team.ID, "name": projectName}).Decode(&project); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Project not found",
			})
		}

		fmt.Printf("[AbortMultipartUpload] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get unique file hashes from commit hash maps in database
	// TODO: Optimize this query
	var fileHashes []string
	cur, err := config.MI.DB.Collection("commits").Find(ctx, bson.M{"project_id": project.ID})
	if err != nil {
		fmt.Printf("[DeleteUnusedStorageObjects] Error while finding all commits for project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	for cur.Next(ctx) {
		var commit models.Commit
		err := cur.Decode(&commit)
		if err != nil {
			fmt.Printf("Error while decoding commit: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
		newFileHashes := lo.Map(lo.Values(commit.Files), func(f models.FileData, _ int) string { return f.Hash })
		fileHashes = append(fileHashes, newFileHashes...)
		fileHashes = lo.Uniq(fileHashes)
	}
	cur.Close(ctx)

	// Search for unused objects in storage
	hasMore := true
	var startAfter *string

	for hasMore {
		prefix := fmt.Sprintf("%s/%s/", team.Name, project.Name)
		res, err := config.SI.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:     &config.SI.ProjectsBucket,
			Prefix:     &prefix,
			StartAfter: startAfter,
		})
		if err != nil {
			fmt.Printf("[DeleteUnusedStorageObjects] Error listing all objects in storage for project: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		hasMore = res.IsTruncated
		startAfter = res.NextContinuationToken
		for _, metadata := range res.Contents {
			if !lo.Contains(fileHashes, strings.Replace(*metadata.Key, prefix, "", 1)) {
				// Object in storage no longer exists in any commit's hash map in the database
				// Delete it
				_, err := config.SI.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: &config.SI.ProjectsBucket,
					Key:    metadata.Key,
				})
				if err != nil {
					fmt.Printf("Error deleting unused object \"%s\" from storage: %v\n", *metadata.Key, err)
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": "Internal server error",
					})
				}
			}
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "All unused files deleted successfully",
	})
}
