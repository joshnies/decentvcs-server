package config

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageInstance struct {
	Client *s3.Client
	Bucket string
	// Size of multipart upload parts in bytes
	MultipartUploadPartSize int64
}

var SI StorageInstance

func getMultipartUploadPartSize() int64 {
	defaultSize := int64(5 * 1024 * 1024) // 5MB
	size := os.Getenv("MULTIPART_UPLOAD_PART_SIZE")
	if size != "" {
		sizeInt, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			panic("Failed to parse MULTIPART_UPLOAD_PART_SIZE environment variable")
		}
		if sizeInt < defaultSize {
			log.Println("MULTIPART_UPLOAD_PART_SIZE too small, using 5MB")
			return defaultSize
		}
		return sizeInt
	}

	// Default to 5MB
	return defaultSize
}

// Initialize storage config instance.
func InitStorage() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		panic("AWS_ACCESS_KEY_ID is not set")
	}

	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		panic("AWS_SECRET_ACCESS_KEY is not set")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		panic("AWS_REGION is not set")
	}

	bucket := os.Getenv("AWS_S3_BUCKET")
	if bucket == "" {
		panic("AWS_S3_BUCKET is not set")
	}

	s3Endpoint := os.Getenv("AWS_S3_ENDPOINT")

	// Initialize S3 client
	var client *s3.Client

	if s3Endpoint != "" {
		// With custom endpoint
		// Used for S3-compatible storage providers other than AWS
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				return aws.Endpoint{
					PartitionID:   "aws",
					URL:           s3Endpoint,
					SigningRegion: region,
				}, nil
			}

			// Fallback to default resolution
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})

		awscfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithEndpointResolverWithOptions(customResolver))
		if err != nil {
			log.Fatalf("failed to load AWS SDK config: %v", err)
		}

		client = s3.NewFromConfig(awscfg)
	} else {
		// With default endpoint (AWS)
		awscfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			log.Fatalf("failed to load AWS SDK config: %v", err)
		}

		client = s3.NewFromConfig(awscfg)
	}

	// Create global storage instance
	SI = StorageInstance{
		Client:                  client,
		Bucket:                  bucket,
		MultipartUploadPartSize: getMultipartUploadPartSize(),
	}
}
