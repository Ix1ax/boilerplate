package files

import (
	"bytes"
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Blobs — реализация Blobs на S3-совместимом хранилище.
type S3Blobs struct {
	client *s3.Client
	bucket string
}

func NewS3Blobs(client *s3.Client, bucket string) *S3Blobs {
	return &S3Blobs{client: client, bucket: bucket}
}

func (b *S3Blobs) Put(ctx context.Context, key, contentType string, data []byte) error {
	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(b.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	return err
}

func (b *S3Blobs) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	object, err := b.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(b.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, 0, err
	}
	return object.Body, aws.ToInt64(object.ContentLength), nil
}

func (b *S3Blobs) Delete(ctx context.Context, key string) error {
	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(b.bucket), Key: aws.String(key)})
	return err
}
