package models

type PresignObjectData struct {
	// If true, the object will be uploaded as multiple parts.
	Multipart bool `json:"multipart"`
	// File size in bytes.
	// Used to determine amount of multipart upload presigned URLs to generate.
	Size int64 `json:"size"`
	// File MIME type
	ContentType string `json:"content_type"`
}

// Request body for `CreateManyPresignedURLs` route.
type PresignManyRequestBody struct {
	Data map[string]PresignObjectData `json:"data"`
}

// Request body for `CreateOnePresignedURL` route.
type PresignOneRequestBody struct {
	Key         string `json:"key"`
	Multipart   bool   `json:"multipart"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// Request body for `CompleteMultipartUpload` route.
type CompleteMultipartUploadRequestBody struct {
	UploadId string `json:"upload_id"`
	Key      string `json:"key"`
	Parts    []struct {
		PartNumber int32  `json:"part_number"`
		ETag       string `json:"etag"`
	} `json:"parts"`
}

// Request body for `AbortMultipartUpload` route.
type AbortMultipartUploadRequestBody struct {
	UploadId string `json:"upload_id"`
	Key      string `json:"key"`
}
