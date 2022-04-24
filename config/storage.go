package config

import (
	"fmt"
	"os"

	"storj.io/uplink"
)

type StorageInstance struct {
	Access *uplink.Access
	Bucket string
}

var SI StorageInstance

func InitStorage() {
	// Create access grant to Storj bucket
	access, err := uplink.ParseAccess(os.Getenv("STORJ_ACCESS_GRANT"))
	if err != nil {
		panic(fmt.Sprintf("Failed to authenticate with Storj: %s", err))
	}

	// Create global storage instance
	SI = StorageInstance{
		Access: access,
		Bucket: os.Getenv("STORJ_BUCKET"),
	}
}
