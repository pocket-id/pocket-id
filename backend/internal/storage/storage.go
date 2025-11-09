package storage

import (
	"context"
	"io"
	"os"
	"time"
)

var (
	TypeFileSystem = "fs"
	TypeS3         = "s3"
)

type ObjectInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
}

type FileStorage interface {
	Save(ctx context.Context, relativePath string, data io.Reader) error
	Open(ctx context.Context, relativePath string) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, relativePath string) error
	DeleteAll(ctx context.Context, prefix string) error
	List(ctx context.Context, prefix string) ([]ObjectInfo, error)
	Walk(ctx context.Context, root string, fn func(ObjectInfo) error) error
	Type() string
}


func IsNotExist(err error) bool {
	return os.IsNotExist(err)
}
