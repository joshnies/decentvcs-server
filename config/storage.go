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

	// TODO: Validate environment variables

	// Create access grant to Storj bucket
	access, err := uplink.RequestAccessWithPassphrase(ctx, os.Getenv("STORJ_SATELLITE"), os.Getenv("STORJ_API_KEY"), os.Getenv("STORJ_API_PASSPHRASE"))
	if err != nil {
		panic(fmt.Sprintf("Failed to authenticate with Storj: %s", err))
	}

	// Create global storage instance
	SI = StorageInstance{
		Access: access,
		Bucket: os.Getenv("STORJ_BUCKET"),
	}
}
