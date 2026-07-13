package s3store

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"

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
		config.WithHTTPClient(&http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout:   15 * time.Second,
				ResponseHeaderTimeout: 45 * time.Second,
				ExpectContinueTimeout: 5 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		}),
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
		// Beget/MinIO and other S3-compatible stores reject aws-chunked trailing checksums.
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
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
		return formatHeadBucketError(c.bucket, err)
	}
	return nil
}

func formatHeadBucketError(bucket string, err error) error {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchBucket":
			return fmt.Errorf(
				"бакет %q не найден: проверьте имя бакета в панели Beget, endpoint (https://s3.ru1.storage.beget.cloud), region (ru-1) и включённый path-style",
				bucket,
			)
		case "Forbidden", "AccessDenied":
			return fmt.Errorf("доступ к бакету %q запрещён: проверьте access key и secret key", bucket)
		}
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "notfound") || strings.Contains(msg, "404") {
		return fmt.Errorf(
			"бакет %q не найден: проверьте имя бакета в панели Beget, endpoint, region и path-style",
			bucket,
		)
	}
	return fmt.Errorf("head bucket: %w", err)
}

func (c *Client) S3() *s3.Client {
	return c.s3
}
