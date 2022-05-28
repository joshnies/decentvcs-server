package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	awstypes "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/config"
	"github.com/joshnies/qc-api/lib/auth"
	"github.com/joshnies/qc-api/lib/storage"
	"github.com/joshnies/qc-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Generate presigned URLs for many objects, scoped to a project.
// These URLs are used by the client to upload files to storage without the need for
// access keys or ACL.
//
// URL params:
//
// - pid: Project ID
//
// - method: Presign method ("PUT" or "GET")
//
// Body: TODO
//
// Returns an array of presigned URLs.
//
func PresignMany(c *fiber.Ctx) error {
	// Validate presign method
	methodStr := strings.ToUpper(c.Params("method"))
	if methodStr != "PUT" && methodStr != "GET" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid presign method. Must be PUT or GET",
		})
	}

	// Get user ID
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse project ID
	pid := c.Params("pid")
	projectObjectId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	project := models.Project{}
	err = config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"_id": projectObjectId, "owner_id": userId}).Decode(&project)
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

	// Parse request body
	var body models.PresignManyRequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Generate presigned URLs
	var method storage.PresignMethod
	if methodStr == "PUT" {
		method = storage.PresignPUT
	} else {
		method = storage.PresignGET
	}

	keyUrlMap, err := storage.PresignMany(c, ctx, storage.PresignManyParams{
		Method: method,
		PID:    pid,
		Data:   body.Data,
	})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Not found",
			})
		}

		fmt.Println(err)
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
// URL params:
//
// - pid: Project ID
//
// - method: Presign method ("PUT" or "GET")
//
// Body: TODO
//
// Returns an array of presigned URLs.
//
func PresignOne(c *fiber.Ctx) error {
	// Validate presign method
	methodStr := strings.ToUpper(c.Params("method"))
	if methodStr != "PUT" && methodStr != "GET" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid presign method. Must be PUT or GET",
		})
	}

	// Get user ID
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse project ID
	pid := c.Params("pid")
	projectObjectId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	project := models.Project{}
	err = config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"_id": projectObjectId, "owner_id": userId}).Decode(&project)
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

	// Parse request body
	var body models.PresignOneRequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Generate presigned URLs
	var method storage.PresignMethod
	if methodStr == "PUT" {
		method = storage.PresignPUT
	} else {
		method = storage.PresignGET
	}

	res, err := storage.PresignOne(c, ctx, storage.PresignOneParams{
		Method:      method,
		PID:         pid,
		Key:         body.Key,
		Multipart:   body.Multipart,
		Size:        body.Size,
		ContentType: body.ContentType,
	})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Not found",
			})
		}

		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(res)
}

// Complete an S3 multipart upload.
// Multipart uploads can be started by generating presigned URLs.
func CompleteMultipartUpload(c *fiber.Ctx) error {
	// Get user ID
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse project ID
	pid := c.Params("pid")
	projectObjectId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	project := models.Project{}
	err = config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"_id": projectObjectId, "owner_id": userId}).Decode(&project)
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

	// Parse request body
	var body models.CompleteMultipartUploadRequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Construct parts
	parts := make([]awstypes.CompletedPart, len(body.Parts))
	for i, part := range body.Parts {
		parts[i] = awstypes.CompletedPart{
			ETag:       &part.ETag,
			PartNumber: part.PartNumber,
		}
	}

	// DEBUG
	// fmt.Printf("Completing upload with ID: \"%s\"\n", body.UploadId)

	// debugRes, err := config.SI.Client.ListMultipartUploads(ctx, &s3.ListMultipartUploadsInput{
	// 	Bucket: &config.SI.Bucket,
	// })
	// if err != nil {
	// 	fmt.Println(err)
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 		"error": "Internal server error",
	// 	})
	// }

	// fmt.Printf("[DEBUG] # of multipart uploads: %d\n", len(debugRes.Uploads))
	// for _, upload := range debugRes.Uploads {
	// 	fmt.Printf("[DEBUG] Upload: %s\n", *upload.UploadId)
	// 	fmt.Printf("[DEBUG]\tKey: %s\n", *upload.Key)
	// 	fmt.Printf("[DEBUG]\tInitiator: %s\n\n", upload.Initiated.Format("2006-01-02 15:04:05"))
	// }
	// ~DEBUG

	// Complete multipart upload
	key := fmt.Sprintf("%s/%s", project.ID, body.Key)
	_, err = config.SI.Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   &config.SI.Bucket,
		Key:      &key,
		UploadId: &body.UploadId,
		MultipartUpload: &awstypes.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		fmt.Println(err)
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
	// Get user ID
	userId, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse project ID
	pid := c.Params("pid")
	projectObjectId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad request",
			"message": "Invalid project ID",
		})
	}

	// Get project
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	project := models.Project{}
	err = config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"_id": projectObjectId, "owner_id": userId}).Decode(&project)
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

	// Parse request body
	var body models.AbortMultipartUploadRequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bad request",
		})
	}

	// Abort multipart upload
	_, err = config.SI.Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   &config.SI.Bucket,
		Key:      &body.Key,
		UploadId: &body.UploadId,
	})
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to abort multipart upload, please make sure the upload ID is correct.",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Success",
	})
}
