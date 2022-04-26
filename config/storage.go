package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"storj.io/uplink"
)

type StorageInstance struct {
	Access *uplink.Access
	Bucket string
}

var SI StorageInstance

func InitStorage() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate environment variables
	accessGrant := os.Getenv("STORJ_ACCESS_GRANT")
	if accessGrant == "" {
		panic("STORJ_ACCESS_GRANT is not set")
	}

	bucket := os.Getenv("STORJ_BUCKET")
	if bucket == "" {
		panic("STORJ_BUCKET is not set")
	}

	// Create access grant to Storj bucket
	access, err := uplink.ParseAccess(accessGrant)
	if err != nil {
		panic(fmt.Sprintf("Failed to authenticate with Storj: %s", err))
	}

	// Open Storj project and ensure bucket exists
	project, err := uplink.OpenProject(ctx, access)
	if err != nil {
		panic(fmt.Sprintf("Failed to open Storj project: %s", err))
	}
	defer project.Close()

	_, err = project.EnsureBucket(ctx, bucket)
	if err != nil {
		panic(fmt.Sprintf("Failed to ensure bucket %s (it probably doesn't exist): %s", bucket, err))
	}

	// Create global storage instance
	SI = StorageInstance{
		Access: access,
		Bucket: bucket,
	}
}
