package filesensor

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// AWSS3Client wraps the AWS SDK v2 S3 client as an [S3ObjectAPI].
type AWSS3Client struct {
	Client *s3.Client
}

// NewAWSS3ClientFromDefaultConfig loads the default AWS config (env, shared config, IAM role).
func NewAWSS3ClientFromDefaultConfig(ctx context.Context) (*AWSS3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	return &AWSS3Client{Client: s3.NewFromConfig(cfg)}, nil
}

// ListObjects implements [S3ObjectAPI].
func (c *AWSS3Client) ListObjects(ctx context.Context, bucket, prefix string) ([]S3ObjectMeta, error) {
	var out []S3ObjectMeta
	var token *string
	for {
		resp, err := c.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: token,
		})
		if err != nil {
			return nil, err
		}
		for _, o := range resp.Contents {
			meta := S3ObjectMeta{Key: aws.ToString(o.Key), Size: aws.ToInt64(o.Size)}
			if o.ETag != nil {
				meta.ETag = aws.ToString(o.ETag)
			}
			if o.LastModified != nil {
				meta.LastModified = *o.LastModified
			}
			out = append(out, meta)
		}
		if !aws.ToBool(resp.IsTruncated) {
			break
		}
		token = resp.NextContinuationToken
	}
	return out, nil
}

// GetObject implements [S3ObjectAPI].
func (c *AWSS3Client) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	resp, err := c.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	return io.ReadAll(resp.Body)
}

// NewS3SourceFromURI builds an [S3Source] using default AWS credentials.
func NewS3SourceFromURI(ctx context.Context, rawURI string) (*S3Source, error) {
	client, err := NewAWSS3ClientFromDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return NewS3Source(rawURI, client)
}
