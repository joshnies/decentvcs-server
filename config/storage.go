package config

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type StorageInstance struct {
	Client *s3.S3
	Bucket string
}

var SI StorageInstance

func InitStorage() {
	// TODO: Validate environment variables

	// Initialize S3 client
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("STORAGE_REGION")),
		Credentials: credentials.NewStaticCredentials(
			os.Getenv("STORAGE_ACCESS_KEY"),
			os.Getenv("STORAGE_SECRET_KEY"),
			"",
		),
		Endpoint: aws.String(os.Getenv("STORAGE_ENDPOINT")),
	}))
	client := s3.New(sess)

	// Assign global instance
	SI = StorageInstance{
		Client: client,
		Bucket: os.Getenv("STORAGE_BUCKET"),
	}

	fmt.Println("Storage initialized ✅")
}
