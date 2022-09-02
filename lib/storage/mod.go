package storage

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/decentvcs/server/config"
	"github.com/decentvcs/server/models"
	"go.mongodb.org/mongo-driver/bson"
)

type PresignMethod string

const (
	PresignMethodGET PresignMethod = "GET"
	PresignMethodPUT PresignMethod = "PUT"
)

func (m PresignMethod) String() string {
	return string(m)
}

// Converts a string to a PresignMethod.
func ToPresignMethod(val string) PresignMethod {
	if val == "PUT" {
		return PresignMethodPUT
	}

	return PresignMethodGET
}

type PresignOptions struct {
	S3PresignClient *s3.PresignClient
	Method          PresignMethod
	Bucket          string
	Key             string
	ContentType     string
	Multipart       bool

	// Only used for multipart uploads
	Size int64

	// Only used for GET presign method
	Team *models.Team
}

type PresignResponse struct {
	UploadID string
	URLs     []string
}

// Returns a presigned URL for fetching or uploading an object from/to storage, respectively.
func Presign(ctx context.Context, opt PresignOptions) (PresignResponse, error) {
	if opt.Method == PresignMethodGET {
		return presignGet(ctx, opt)
	} else if opt.Method == PresignMethodPUT {
		return presignPut(ctx, opt)
	}
	return PresignResponse{}, errors.New("presign method must be GET or PUT")
}

// Returns a presigned GET URL for fetching an object from storage.
func presignGet(ctx context.Context, opt PresignOptions) (PresignResponse, error) {
	res, err := opt.S3PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &opt.Bucket,
		Key:    &opt.Key,
	})
	if err != nil {
		return PresignResponse{}, err
	}

	// Get object size
	s3Res, err := config.SI.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &opt.Bucket,
		Key:    &opt.Key,
	})
	if err != nil {
		return PresignResponse{}, err
	}

	// Update team bandwidth usage
	newBandwidth := opt.Team.BandwidthUsedMB + float64(s3Res.ContentLength)/1024/1024
	if _, err = config.MI.DB.Collection("teams").UpdateOne(
		ctx,
		bson.M{"_id": opt.Team.ID},
		bson.M{
			"$set": bson.M{
				"bandwidth_used_mb": newBandwidth,
			},
		},
	); err != nil {
		return PresignResponse{}, err
	}

	return PresignResponse{
		URLs: []string{res.URL},
	}, nil
}

// Returns a presigned PUT URL for uploading an object to storage.
func presignPut(ctx context.Context, opt PresignOptions) (PresignResponse, error) {
	var uploadID string
	urls := []string{}

	if opt.Multipart {
		// Multipart upload
		expiresAt := time.Now().Add(time.Hour * 24) // 24 hours
		multipartRes, err := config.SI.Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
			Bucket:      &opt.Bucket,
			Key:         &opt.Key,
			ContentType: &opt.ContentType,
			Expires:     &expiresAt,
		})
		if err != nil {
			return PresignResponse{}, err
		}

		uploadID = *multipartRes.UploadId
		remaining := opt.Size
		var partNum int32 = 1
		var currentSize int64
		for remaining != 0 {
			// Determine current part size
			if remaining < config.SI.MultipartUploadPartSize {
				currentSize = remaining
			} else {
				currentSize = config.SI.MultipartUploadPartSize
			}

			// Generate presigned URL
			res, err := opt.S3PresignClient.PresignUploadPart(ctx, &s3.UploadPartInput{
				Bucket:        &opt.Bucket,
				Key:           &opt.Key,
				UploadId:      multipartRes.UploadId,
				PartNumber:    partNum,
				ContentLength: currentSize,
			})
			if err != nil {
				return PresignResponse{}, err
			}

			// Add presigned URL to result slice
			urls = append(urls, res.URL)

			// Update remaining size and part number
			remaining -= currentSize
			partNum++
		}
	} else {
		// Single upload
		res, err := opt.S3PresignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: &opt.Bucket,
			Key:    &opt.Key,
		})
		if err != nil {
			return PresignResponse{}, err
		}

		urls = append(urls, res.URL)
	}

	return PresignResponse{
		UploadID: uploadID,
		URLs:     urls,
	}, nil
}
