package cloudflare

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	config2 "foro-unsaac-backend/internal/config"
	"foro-unsaac-backend/internal/domain"
)

type r2Repository struct {
	s3Client      *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
	publicDomain  string
}

func NewR2Repository(storageConfig config2.StorageConfig) (domain.StorageRepository, error) {
	// R2 usa el endpoint con el Account ID de Cloudflare
	r2Endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", storageConfig.AccountID)

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("auto"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               r2Endpoint,
					HostnameImmutable: true,
					SigningRegion:     "auto",
				}, nil
			},
		)),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				storageConfig.AccessKey, storageConfig.SecretKey, "")),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(client)

	return &r2Repository{
		s3Client:      client,
		presignClient: presignClient,
		bucketName:    storageConfig.Bucket,
		publicDomain:  storageConfig.PublicDomain,
	}, nil
}

func (r *r2Repository) UploadFile(ctx context.Context, prefix string, filename string, content io.Reader, contentType string) (string, error) {
	key := fmt.Sprintf("%s/%s", prefix, filename)

	_, err := r.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        content,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	return fmt.Sprintf("%s/%s", r.publicDomain, key), nil
}

func (r *r2Repository) DeleteFile(ctx context.Context, key string) error {
	_, err := r.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	return err
}

func (r *r2Repository) GetPresignedUploadURL(ctx context.Context, key string, contentType string) (string, error) {
	req, err := r.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(15*time.Minute)) // Expira rápido por seguridad

	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (r *r2Repository) MoveObject(ctx context.Context, sourceKey string, destinationKey string) error {
	// 1. Copiar el objeto a la nueva ubicación dentro del mismo bucket
	// Nota: El formato de CopySource requiere incluir el nombre del bucket al inicio
	copySource := fmt.Sprintf("%s/%s", r.bucketName, sourceKey)

	_, err := r.s3Client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(r.bucketName),
		CopySource: aws.String(copySource),
		Key:        aws.String(destinationKey),
	})
	if err != nil {
		return fmt.Errorf("failed to copy object in R2: %w", err)
	}

	// 2. Borrar el objeto original de la ubicación vieja
	_, err = r.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(sourceKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete source object after copy in R2: %w", err)
	}

	return nil
}

func (r *r2Repository) GetPresignedPutURL(ctx context.Context, key string, expiresInSeconds int) (string, error) {
	req, err := r.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(time.Duration(expiresInSeconds)*time.Second))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (r *r2Repository) DownloadFile(ctx context.Context, key string, destPath string) error {
	result, err := r.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download from R2: %w", err)
	}
	defer result.Body.Close()

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, result.Body)
	return err
}

// DownloadFileBytes streams the object into memory. Bounded by R2's
// get-object response — the caller caps via io.LimitReader before
// passing data to a processor. Use only for small files (avatars).
//
// Accepts either a raw key ("users/{id}/avatar/x.jpg") OR a full URL
// ("https://assets.unsaac.com/users/{id}/avatar/x.jpg") — the latter
// is the public CDN URL form; we strip the domain prefix here so the
// caller can pass whichever artifact they have on hand.
func (r *r2Repository) DownloadFileBytes(ctx context.Context, keyOrURL string) ([]byte, error) {
	key := normalizeKey(keyOrURL, r.publicDomain)

	result, err := r.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", key, err)
	}
	defer result.Body.Close()

	// Cap at 16 MB to bound memory use — well above the 8 MB cap on the
	// upload path. Anything larger is a bug at the call site.
	return io.ReadAll(io.LimitReader(result.Body, 16<<20))
}

// normalizeKey strips the public-domain prefix from a CDN URL when
// present, returning just the R2 key path. If `key` doesn't start
// with the public domain it's returned verbatim.
func normalizeKey(key, publicDomain string) string {
	if publicDomain == "" {
		return key
	}
	if strings.HasPrefix(key, publicDomain+"/") {
		return strings.TrimPrefix(key, publicDomain+"/")
	}
	return key
}

// AvatarVariantURL composes the canonical public URL for an avatar
// variant. The version query parameter is the cache-busting mechanism.
func (r *r2Repository) AvatarVariantURL(ctx context.Context, userID string, variant string, version int, ext string) string {
	return fmt.Sprintf("%s/users/%s/avatar/%s.%s?v=%d", r.publicDomain, userID, variant, ext, version)
}

func (r *r2Repository) UploadDirectory(ctx context.Context, localDir string, remotePrefix string) (string, error) {
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		localPath := filepath.Join(localDir, entry.Name())
		key := fmt.Sprintf("%s/%s", remotePrefix, entry.Name())

		content, err := os.Open(localPath)
		if err != nil {
			return "", fmt.Errorf("failed to open file: %w", err)
		}

		_, err = r.s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(r.bucketName),
			Key:         aws.String(key),
			Body:        content,
			ContentType: aws.String(contentTypeFor(entry.Name())),
		})
		content.Close()
		if err != nil {
			return "", fmt.Errorf("failed to upload %s: %w", key, err)
		}
	}

	return fmt.Sprintf("%s/%s/index.m3u8", r.publicDomain, remotePrefix), nil
}

// contentTypeFor returns the appropriate Content-Type for HLS-related files.
// Falls back to application/octet-stream for unknown extensions.
func contentTypeFor(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".m3u8":
		return "application/vnd.apple.mpegurl"
	case ".ts":
		return "video/mp2t"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}
