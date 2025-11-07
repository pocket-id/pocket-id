package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type ObjectInfo struct {
	Path string
	Size int64
}

type FileStorage interface {
	Save(ctx context.Context, relativePath string, data io.Reader) error
	Open(ctx context.Context, relativePath string) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, relativePath string) error
	DeleteAll(ctx context.Context, prefix string) error
	List(ctx context.Context, prefix string) ([]ObjectInfo, error)
}

type Config struct {
	Backend           string
	Root              string
	S3Bucket          string
	S3Region          string
	S3Endpoint        string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3ForcePathStyle  bool
}

// NewFileStorage initializes the configured storage backend.
func NewFileStorage(ctx context.Context, cfg Config) (FileStorage, error) {
	switch cfg.Backend {
	case "", "filesystem":
		if strings.TrimSpace(cfg.Root) == "" {
			return nil, errors.New("filesystem storage requires a root path")
		}
		return newFilesystemStorage(cfg.Root)
	case "s3":
		if cfg.S3Bucket == "" || cfg.S3Region == "" {
			return nil, errors.New("s3 storage requires both bucket and region")
		}
		return newS3Storage(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported file backend '%s'", cfg.Backend)
	}
}

func IsNotExist(err error) bool {
	return os.IsNotExist(err)
}
