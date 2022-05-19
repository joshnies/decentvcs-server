package storage

import (
	"context"
	"fmt"
	"sync"

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
	Keys   []string
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
func PresignMany(fctx *fiber.Ctx, ctx context.Context, params PresignManyParams) (map[string]string, error) {
	client := s3.NewPresignClient(config.SI.Client)

	// Destructure params
	method := params.Method
	pid := params.PID
	keys := params.Keys

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
	keyUrlMap := make(map[string]string)

	for _, localKey := range keys {
		projectKey := fmt.Sprintf("%s/%s", pid, localKey)

		if method == PresignPUT {
			// PUT
			res, err := client.PresignPutObject(ctx, &s3.PutObjectInput{
				Bucket: &config.SI.Bucket,
				Key:    &projectKey,
			})
			if err != nil {
				panic(err)
			}

			keyUrlMap[localKey] = res.URL
		} else {
			// GET
			res, err := client.PresignGetObject(ctx, &s3.GetObjectInput{
				Bucket: &config.SI.Bucket,
				Key:    &projectKey,
			})
			if err != nil {
				panic(err)
			}

			keyUrlMap[localKey] = res.URL
		}
	}

	return keyUrlMap, nil
}
