package config

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageInstance struct {
	Client *s3.Client
	Bucket string
}

var SI StorageInstance

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

	bucket := os.Getenv("AWS_S3_BUCKET")
	if bucket == "" {
		panic("AWS_S3_BUCKET is not set")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		panic("AWS_REGION is not set")
	}

	s3Endpoint := os.Getenv("AWS_S3_ENDPOINT")
	if s3Endpoint == "" {
		panic("AWS_S3_ENDPOINT is not set")
	}

	// Initialize S3 client
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				PartitionID: "aws",
				// URL:           "https://s3.filebase.com",
				URL: s3Endpoint,
				// SigningRegion: "us-east-1",
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

	client := s3.NewFromConfig(awscfg)

	// Create global storage instance
	SI = StorageInstance{
		Client: client,
		Bucket: bucket,
	}
}
