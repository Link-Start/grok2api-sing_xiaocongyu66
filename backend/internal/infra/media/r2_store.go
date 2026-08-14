package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// R2Config configures Cloudflare R2 (S3-compatible) object storage.
type R2Config struct {
	// Endpoint is the S3 API endpoint, e.g. https://<accountid>.r2.cloudflarestorage.com
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	// Region is required by the SDK; R2 accepts "auto".
	Region string
	// Prefix is an optional object key prefix (no leading slash), e.g. "grok2api".
	Prefix string
	// PublicBaseURL optional CDN/custom domain for direct object URLs (not used for gateway serving).
	PublicBaseURL string
}

// R2Store stores media objects in Cloudflare R2 via the S3 API.
type R2Store struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewR2Store builds an R2-backed MediaObjectStorage.
func NewR2Store(cfg R2Config) (*R2Store, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	accessKey := strings.TrimSpace(cfg.AccessKeyID)
	secretKey := strings.TrimSpace(cfg.SecretAccessKey)
	bucket := strings.TrimSpace(cfg.Bucket)
	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, fmt.Errorf("media.r2 需要 endpoint、accessKeyId、secretAccessKey、bucket")
	}
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		return nil, fmt.Errorf("media.r2.endpoint 无效: %w", err)
	}
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}
	prefix := strings.Trim(strings.TrimSpace(cfg.Prefix), "/")

	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(endpoint),
		Region:       region,
		Credentials:  credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		// R2 is path-style friendly and does not use AWS virtual-host addressing the same way.
		UsePathStyle: true,
	})
	return &R2Store{client: client, bucket: bucket, prefix: prefix}, nil
}

func (s *R2Store) SaveImage(ctx context.Context, id, mimeType string, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	extension, ok := imageExtension(mimeType)
	if !ok || len(id) < 2 {
		return "", fmt.Errorf("图片存储参数无效")
	}
	// Logical key matches LocalStore layout so metadata stays portable.
	storageKey := path.Join("images", id[:2], id+extension)
	objectKey := s.objectKey(storageKey)
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(mimeType),
		// Immutable content-addressed objects.
		CacheControl: aws.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return "", fmt.Errorf("上传图片到 R2: %w", err)
	}
	return storageKey, nil
}

func (s *R2Store) Open(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key, err := s.validatedKey(storageKey)
	if err != nil {
		return nil, err
	}
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, os.ErrNotExist
		}
		// aws-sdk may wrap not-found differently
		if strings.Contains(strings.ToLower(err.Error()), "nosuchkey") || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("从 R2 读取媒体: %w", err)
	}
	return out.Body, nil
}

func (s *R2Store) Delete(ctx context.Context, storageKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key, err := s.validatedKey(storageKey)
	if err != nil {
		return err
	}
	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("从 R2 删除媒体: %w", err)
	}
	return nil
}

func (s *R2Store) objectKey(storageKey string) string {
	clean := strings.Trim(strings.ReplaceAll(storageKey, "\\", "/"), "/")
	if s.prefix == "" {
		return clean
	}
	return s.prefix + "/" + clean
}

func (s *R2Store) validatedKey(storageKey string) (string, error) {
	clean := path.Clean("/" + strings.ReplaceAll(strings.TrimSpace(storageKey), "\\", "/"))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", fmt.Errorf("媒体存储路径无效")
	}
	return s.objectKey(clean), nil
}
