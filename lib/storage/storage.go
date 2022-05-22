package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/config"
	"github.com/joshnies/qc-api/lib/auth"
	"github.com/joshnies/qc-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PresignMethod int

const (
	PresignPUT PresignMethod = iota
	PresignGET
)

type PresignManyParams struct {
	Method PresignMethod
	PID    string
	// Map of object key (typically the file hash) to an object containing info about the file's upload.
	Data map[string]models.PresignObjectData
}

type PresignRoutineParams struct {
	Method     PresignMethod
	Client     *s3.PresignClient
	LocalKey   string
	ProjectKey string
	KeyURLMap  map[string]string
	WG         *sync.WaitGroup
}

// Generate many presigned URLs for the specified objects, scoped to a project.
//
// Params:
//
// - fctx: Fiber context
//
// - ctx: Go context
//
// - params: Additional params for presigning
//
// Returns map of keys to presigned URLs.
//
func PresignMany(fctx *fiber.Ctx, ctx context.Context, params PresignManyParams) (map[string][]string, error) {
	client := s3.NewPresignClient(config.SI.Client)

	// Destructure params
	method := params.Method
	pid := params.PID

	// Create project object ID
	projectObjId, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		return nil, err
	}

	// Make sure user has access to the project
	userId, err := auth.GetUserID(fctx)
	if err != nil {
		return nil, err
	}

	var project models.Project
	err = config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"_id": projectObjId, "owner_id": userId}).Decode(&project)
	if err != nil {
		return nil, err
	}

	// Generate presigned URLs
	keyUrlMap := make(map[string][]string)

	for localKey, data := range params.Data {
		projectKey := fmt.Sprintf("%s/%s", pid, localKey)

		if method == PresignPUT {
			// PUT
			if data.Multipart {
				// Multipart upload
				contentType := data.ContentType
				expiresAt := time.Now().Add(time.Hour * 24) // 24 hours
				multipartRes, err := config.SI.Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket:      &config.SI.Bucket,
					Key:         &projectKey,
					ContentType: &contentType,
					Expires:     &expiresAt,
				})
				if err != nil {
					return nil, err
				}

				keyUrlMap[localKey] = []string{}
				remaining := data.Size
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
						Key:           &projectKey,
						UploadId:      multipartRes.UploadId,
						PartNumber:    partNum,
						ContentLength: config.SI.MultipartUploadPartSize,
					})
					if err != nil {
						return nil, err
					}

					// Add presigned URL to map
					keyUrlMap[localKey] = append(keyUrlMap[localKey], res.URL)

					// Update remaining size and part number
					remaining -= currentSize
					partNum++
				}
			} else {
				// Single upload
				res, err := client.PresignPutObject(ctx, &s3.PutObjectInput{
					Bucket: &config.SI.Bucket,
					Key:    &projectKey,
				})
				if err != nil {
					panic(err)
				}

				keyUrlMap[localKey] = []string{res.URL}
			}
		} else {
			// GET
			res, err := client.PresignGetObject(ctx, &s3.GetObjectInput{
				Bucket: &config.SI.Bucket,
				Key:    &projectKey,
			})
			if err != nil {
				panic(err)
			}

			keyUrlMap[localKey] = []string{res.URL}
		}
	}

	return keyUrlMap, nil
}
