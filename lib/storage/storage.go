package storage

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joshnies/qc-api/config"
)

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
func PresignMany(ctx context.Context, pid string, keys []string) (map[string]string, error) {
	client := s3.NewPresignClient(config.SI.Client)

	// Get presigned URLs
	keyUrlMap := make(map[string]string)
	for _, k := range keys {
		key := fmt.Sprintf("%s/%s", pid, k)
		res, err := client.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: &config.SI.Bucket,
			Key:    &key,
		})
		if err != nil {
			return nil, err
		}

		keyUrlMap[k] = res.URL
	}

	return keyUrlMap, nil
}
