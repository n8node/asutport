package s3store

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appconfig "github.com/n8node/asutport/internal/config"
)

type Client struct {
	bucket string
	s3     *s3.Client
}

func NewClient(cfg *appconfig.Config) (*Client, error) {
	if !cfg.S3Configured() {
		return nil, fmt.Errorf("s3 credentials are not configured")
	}

	endpoint := strings.TrimSuffix(strings.TrimSpace(cfg.S3Endpoint), "/")
	region := strings.TrimSpace(cfg.S3Region)
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.S3AccessKey,
			cfg.S3SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = cfg.S3UsePathStyle
	})

	return &Client{
		bucket: cfg.S3Bucket,
		s3:     client,
	}, nil
}

func (c *Client) Bucket() string {
	return c.bucket
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return fmt.Errorf("head bucket: %w", err)
	}
	return nil
}

func (c *Client) S3() *s3.Client {
	return c.s3
}
