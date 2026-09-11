// Package s3 wraps object storage for product images. Clients upload directly to
// S3 with a presigned URL; the service only issues URLs and tracks keys.
//
// Key layout (one prefix per business, then per product):
//
//	products/<tenant_id>/<product_id>/<image_id><ext>
package s3

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"

	appconfig "github.com/jayedbinnazir/product-service/internal/config"
)

// AllowedContentTypes maps an accepted upload content-type to a file extension.
var AllowedContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/avif": ".avif",
}

type Storage struct {
	client        *s3.Client
	presign       *s3.PresignClient
	bucket        string
	publicBaseURL string // CDN or bucket URL, no trailing slash
}

// New builds the storage client. It does not make a network call, so a missing
// bucket / credentials only surfaces when an image operation actually runs.
func New(ctx context.Context, cfg appconfig.S3Config) (*Storage, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(orDefault(cfg.Region, "us-east-1")),
	}
	if cfg.AccessKeyID != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" { // MinIO / LocalStack in dev
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	})

	base := cfg.PublicBaseURL
	if base == "" && cfg.Endpoint != "" {
		base = strings.TrimRight(cfg.Endpoint, "/") + "/" + cfg.Bucket
	}

	return &Storage{
		client:        client,
		presign:       s3.NewPresignClient(client),
		bucket:        cfg.Bucket,
		publicBaseURL: strings.TrimRight(base, "/"),
	}, nil
}

// NewImageKey builds the object key for a not-yet-uploaded product image.
func (s *Storage) NewImageKey(tenantID, productID, imageID uuid.UUID, contentType string) (key, ext string, ok bool) {
	ext, ok = AllowedContentTypes[contentType]
	if !ok {
		return "", "", false
	}
	return fmt.Sprintf("products/%s/%s/%s%s", tenantID, productID, imageID, ext), ext, true
}

// KeyBelongsTo reports whether key is under this tenant+product's prefix (guards
// the confirm step against attaching someone else's object).
func (s *Storage) KeyBelongsTo(key string, tenantID, productID uuid.UUID) bool {
	return strings.HasPrefix(key, fmt.Sprintf("products/%s/%s/", tenantID, productID))
}

// PresignPut returns a URL the client PUTs the raw file bytes to.
func (s *Storage) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	out, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

// Exists checks the object is really in the bucket (called on confirm).
func (s *Storage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
		return false, nil
	}
	return false, err
}

// Delete removes an object. A missing object is not an error.
func (s *Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// PublicURL is the browser URL for a stored key.
func (s *Storage) PublicURL(key string) string {
	return s.publicBaseURL + "/" + key
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
