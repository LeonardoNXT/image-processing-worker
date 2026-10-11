package ports

import (
	"context"
	"io"
)

type ObjectStorage interface {
	Get(context.Context, string) (io.ReadCloser, error)
	Put(ctx context.Context, key string, content io.Reader, size int64, contentType string) error
	Delete(context.Context, string) error
}
