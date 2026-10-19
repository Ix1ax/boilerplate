package events

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// StoryS3 использует отдельный приватный бакет. Доступ к объекту выдаёт только API истории.
type StoryS3 struct {
	client *s3.Client
	bucket string
}

func NewStoryS3(endpoint, region, accessKey, secretKey, bucket string) *StoryS3 {
	client := s3.New(s3.Options{Region: region, BaseEndpoint: aws.String(endpoint), UsePathStyle: true, Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""), RetryMaxAttempts: 2, HTTPClient: &http.Client{Timeout: 20 * time.Second}})
	return &StoryS3{client: client, bucket: bucket}
}
func (b *StoryS3) Put(ctx context.Context, key, contentType string, data []byte) error {
	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(b.bucket), Key: aws.String(key), ContentType: aws.String(contentType), Body: bytes.NewReader(data)})
	return err
}
func (b *StoryS3) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(b.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, 0, err
	}
	return out.Body, aws.ToInt64(out.ContentLength), nil
}
func (b *StoryS3) Delete(ctx context.Context, key string) error {
	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(b.bucket), Key: aws.String(key)})
	return err
}

// EnsureBucket не маскирует ошибку доступа под отсутствие бакета.
func (b *StoryS3) EnsureBucket(ctx context.Context) error {
	_, err := b.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(b.bucket)})
	if err == nil {
		return nil
	}
	var response *smithyhttp.ResponseError
	if !errors.As(err, &response) || response.HTTPStatusCode() != 404 {
		return err
	}
	_, err = b.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(b.bucket)})
	var api smithy.APIError
	if errors.As(err, &api) && api.ErrorCode() == "BucketAlreadyOwnedByYou" {
		return nil
	}
	return err
}

func (b *StoryS3) Check(ctx context.Context) error {
	_, err := b.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(b.bucket)})
	return err
}
