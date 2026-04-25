package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Adapter struct {
	client *s3.Client
	bucket string
}

type Config struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	UsePathStyle    bool
}

func NewAdapter(ctx context.Context, cfg Config) (*Adapter, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	_, _ = client.CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket: aws.String("platform-documents"),
	})

	return &Adapter{
		client: client,
		bucket: cfg.BucketName,
	}, nil
}

func (a *Adapter) Upload(ctx context.Context, tenantID, filename string, reader io.Reader, size int64) (string, error) {
	objectKey := fmt.Sprintf("%s/%s", tenantID, filename)

	input := &s3.PutObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(objectKey),
		Body:   reader,
	}

	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}

	_, err := a.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("s3 put object: %w", err)
	}

	return fmt.Sprintf("s3://%s/%s", a.bucket, objectKey), nil
}

func (a *Adapter) Download(ctx context.Context, uri string) (io.ReadCloser, error) {
	prefix := fmt.Sprintf("s3://%s/", a.bucket)
	if len(uri) <= len(prefix) {
		return nil, fmt.Errorf("invalid s3 uri format: %s", uri)
	}
	objectKey := uri[len(prefix):]

	input := &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(objectKey),
	}

	output, err := a.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("s3 get object: %w", err)
	}

	return output.Body, nil
}
