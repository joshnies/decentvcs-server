package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/joshnies/qc-api/config"
	"github.com/joshnies/qc-api/lib/auth"
	"github.com/joshnies/qc-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
// - pid: Project ID
//
// - keys: Object keys
//
// Returns map of keys to presigned URLs.
//
func PresignMany(fctx *fiber.Ctx, ctx context.Context, params PresignManyParams) (map[string]string, error) {
	client := s3.NewPresignClient(config.SI.Client)

	// Destructure params
	method := params.Method
	pid := params.PID
	keys := params.Keys

	// Make sure user has access to the project
	userId, err := auth.GetUserID(fctx)
	if err != nil {
		return nil, err
	}

	var project models.Project
	err = config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"project_id": pid, "owner_id": userId}).Decode(&project)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("unauthorized")
		}

		return nil, err
	}

	// Generate presigned URLs in parallel
	keyUrlMap := make(map[string]string)

	var wg sync.WaitGroup
	wg.Add(len(keys))

	for _, localKey := range keys {
		// Generate presigned URL
		projectKey := fmt.Sprintf("%s/%s", pid, localKey)
		go presignRoutine(ctx, PresignRoutineParams{
			Method:     method,
			Client:     client,
			LocalKey:   localKey,
			ProjectKey: projectKey,
			KeyURLMap:  keyUrlMap,
			WG:         &wg,
		})
	}

	wg.Wait()
	return keyUrlMap, nil
}

func presignRoutine(ctx context.Context, params PresignRoutineParams) {
	defer params.WG.Done()

	if params.Method == PresignPUT {
		// PUT
		res, err := params.Client.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: &config.SI.Bucket,
			Key:    &params.ProjectKey,
		})
		if err != nil {
			panic(err)
		}

		params.KeyURLMap[params.LocalKey] = res.URL
		return
	}

	// GET
	res, err := params.Client.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &config.SI.Bucket,
		Key:    &params.ProjectKey,
	})
	if err != nil {
		panic(err)
	}

	params.KeyURLMap[params.LocalKey] = res.URL
}
