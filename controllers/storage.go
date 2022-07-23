package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	awstypes "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/lib/team_lib"
	"github.com/joshnies/decent-vcs/models"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type PresignMethod int

const (
	PresignMethodPUT PresignMethod = iota
	PresignMethodGET
)

func FormatStorageKey(team models.Team, project models.Project, key string) string {
	return fmt.Sprintf("%s/%s/%s", team.Name, project.Name, key)
}

// Generate presigned GET URLs for many objects, scoped to a project.
// These URLs are used by the client to upload files to storage without the need for
// access keys or ACL.
//
// Use `PresignOne` with the `PUT` method argument if you need to create PUT URLs.
//
// Returns an array of presigned URLs.
//
func PresignManyGET(c *fiber.Ctx) error {
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

		fmt.Printf("[PresignManyGET] Error getting project: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Parse request body
	var body models.PresignManyRequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Generate presigned URLs
	client := s3.NewPresignClient(config.SI.Client)

	// Generate presigned URLs
	keyUrlMap := make(map[string]string)

	for _, localKey := range body.Keys {
		remoteKey := FormatStorageKey(*team, project, localKey)
		res, err := client.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: &config.SI.Bucket,
			Key:    &remoteKey,
		})
		if err != nil {
			fmt.Printf("[PresignManyGET] Error generating presigned URL: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		keyUrlMap[localKey] = res.URL

		// Get object size
		s3Res, err := config.SI.Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: &config.SI.Bucket,
			Key:    &remoteKey,
		})
		if err != nil {
			fmt.Printf("[PresignOne] Error getting object size: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		// Add object size to team bandwidth usage
		team.BandwidthUsedMB += float64(s3Res.ContentLength) / 1024 / 1024
	}

	// Update team bandwidth usage in database
	if _, err := config.MI.DB.Collection("teams").UpdateOne(
		ctx,
		bson.M{"_id": team.ID},
		bson.M{
			"$set": bson.M{
				"bandwidth_used_mb": team.BandwidthUsedMB,
			},
		},
	); err != nil {
		fmt.Printf("[PresignManyGET] Error updating team bandwidth usage: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(keyUrlMap)
}

// Generate presigned URL(s) for the specified storage object, scoped to a project.
// These URLs are used by the client to upload files to storage without the need for
// access keys or ACL.
//
// Returns an array of presigned URLs.
//
func PresignOne(c *fiber.Ctx) error {
	team := team_lib.GetTeamFromContext(c)
	projectName := c.Params("project_name")

	// Validate presign method
	methodStr := strings.ToUpper(c.Params("method"))
	if methodStr != "PUT" && methodStr != "GET" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid presign method. Must be PUT or GET",
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

	var method PresignMethod
	if methodStr == "PUT" {
		method = PresignMethodPUT
	} else if methodStr == "GET" {
		method = PresignMethodGET
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid presign method. Must be \"PUT\" or \"GET\".",
		})
	}

	client := s3.NewPresignClient(config.SI.Client)
	remoteKey := FormatStorageKey(*team, project, body.Key)
	var uploadId string
	urls := []string{}

	if method == PresignMethodPUT {
		// PUT
		if body.Multipart {
			// Multipart upload
			contentType := body.ContentType
			expiresAt := time.Now().Add(time.Hour * 24) // 24 hours
			multipartRes, err := config.SI.Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
				Bucket:      &config.SI.Bucket,
				Key:         &remoteKey,
				ContentType: &contentType,
				Expires:     &expiresAt,
			})
			if err != nil {
				fmt.Println(err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			uploadId = *multipartRes.UploadId
			remaining := body.Size
			var partNum int32 = 1
			var currentSize int64
			for remaining != 0 {
				// Determine current part size
				if remaining < config.SI.MultipartUploadPartSize {
					currentSize = remaining
				} else {
					currentSize = config.SI.MultipartUploadPartSize
				}

				// Generate presigned URL
				res, err := client.PresignUploadPart(ctx, &s3.UploadPartInput{
					Bucket:        &config.SI.Bucket,
					Key:           &remoteKey,
					UploadId:      multipartRes.UploadId,
					PartNumber:    partNum,
					ContentLength: currentSize,
				})
				if err != nil {
					fmt.Printf("[PresignOne] Error presigning PUT URL for an upload part: %v\n", err)
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": "Internal server error",
					})
				}

				// Add presigned URL to result slice
				urls = append(urls, res.URL)

				// Update remaining size and part number
				remaining -= currentSize
				partNum++
			}
		} else {
			// Single upload
			res, err := client.PresignPutObject(ctx, &s3.PutObjectInput{
				Bucket: &config.SI.Bucket,
				Key:    &remoteKey,
			})
			if err != nil {
				fmt.Printf("[PresignOne] Error presigning PUT URL for a single object: %v\n", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}

			urls = append(urls, res.URL)
		}
	} else {
		// GET
		//
		// Generate presigned URL
		res, err := client.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: &config.SI.Bucket,
			Key:    &remoteKey,
		})
		if err != nil {
			fmt.Printf("[PresignOne] Error presigning GET URL for an object: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		urls = append(urls, res.URL)

		// Get object size
		s3Res, err := config.SI.Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: &config.SI.Bucket,
			Key:    &remoteKey,
		})
		if err != nil {
			fmt.Printf("[PresignOne] Error getting object size: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		// Update team bandwidth usage
		team.BandwidthUsedMB += float64(s3Res.ContentLength) / 1024 / 1024
		if _, err = config.MI.DB.Collection("teams").UpdateOne(
			ctx,
			bson.M{"_id": team.ID},
			bson.M{
				"$set": bson.M{
					"bandwidth_used_mb": team.BandwidthUsedMB,
				},
			},
		); err != nil {
			fmt.Printf("[PresignOne] Error updating team bandwidth usage: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	return c.JSON(models.PresignOneResponse{
		UploadID: uploadId,
		URLs:     urls,
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
	key := FormatStorageKey(*team, project, body.Key)
	if _, err := config.SI.Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   &config.SI.Bucket,
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
		Bucket:   &config.SI.Bucket,
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
		fileHashes = append(fileHashes, lo.Values(commit.HashMap)...)
		fileHashes = lo.Uniq(fileHashes)
	}
	cur.Close(ctx)

	// Search for unused objects in storage
	hasMore := true
	var startAfter *string

	for hasMore {
		prefix := fmt.Sprintf("%s/%s/", team.Name, project.Name)
		res, err := config.SI.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:     &config.SI.Bucket,
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
					Bucket: &config.SI.Bucket,
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
