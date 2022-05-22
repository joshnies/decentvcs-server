package models

type PresignObjectData struct {
	// If true, the object will be uploaded as multiple parts.
	Multipart bool
	// File size in bytes.
	// Used to determine amount of multipart upload presigned URLs to generate.
	Size int64
	// File MIME type
	ContentType string
}

type PresignedURLRequestBody struct {
	Data map[string]PresignObjectData `json:"keys"`
}
