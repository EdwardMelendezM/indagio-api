package domain

import (
	"context"
	"io"
)

// StorageRepository define el contrato abstracto para interactuar con R2/S3
type StorageRepository interface {
	UploadFile(ctx context.Context, prefix string, filename string, content io.Reader, contentType string) (string, error)
	DeleteFile(ctx context.Context, key string) error
	GetPresignedUploadURL(ctx context.Context, key string, contentType string) (string, error)
	GetPresignedPutURL(ctx context.Context, key string, expiresInSeconds int) (string, error)
	MoveObject(ctx context.Context, sourceKey string, destinationKey string) error
	DownloadFile(ctx context.Context, key string, destPath string) error
	// DownloadFileBytes streams the object into memory. The caller is
	// responsible for capping the size before invoking (e.g. via io.LimitReader
	// in the imaging package); R2 objects larger than memory will crash.
	// Use only for small files (avatars, previews) — never for video.
	DownloadFileBytes(ctx context.Context, key string) ([]byte, error)
	// AvatarVariantURL builds a stable public URL for an avatar variant
	// (e.g. "thumb", "medium", "full") under the user's avatar folder.
	// The version is appended as a query parameter for cache-busting.
	AvatarVariantURL(ctx context.Context, userID string, variant string, version int, ext string) string
	UploadDirectory(ctx context.Context, localDir string, remotePrefix string) (string, error)
}
