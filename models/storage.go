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

// Request body for `PresignMany` route.
type PresignManyRequest []struct {
	Method      string `json:"method"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Multipart   bool   `json:"multipart"`
	Size        int64  `json:"size"`
}

// Request body for `PresignOne` route.
type PresignOneRequest struct {
	Method      string `json:"method"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Multipart   bool   `json:"multipart"`
	Size        int64  `json:"size"`
}

// Response body for `PresignOne` and `PresignMany` routes.
type PresignResponse struct {
	// Presigned URLs.
	URLs []string `json:"urls"`
	// ID of the multipart upload. Only present if `multipart` is true and method is `PUT`.
	UploadID string `json:"upload_id"`
}

type MultipartUploadPart struct {
	PartNumber int32  `json:"part_number"`
	ETag       string `json:"etag"`
}

// Request body for `CompleteMultipartUpload` route.
type CompleteMultipartUploadRequestBody struct {
	UploadId string                `json:"upload_id"`
	Key      string                `json:"key"`
	Parts    []MultipartUploadPart `json:"parts"`
}

// Request body for `AbortMultipartUpload` route.
type AbortMultipartUploadRequestBody struct {
	UploadId string `json:"upload_id"`
	Key      string `json:"key"`
}
