package minio

import (
	"context"
	"image-processing-worker/infrastructure/config"
	"io"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func Client(c *config.Config) (*miniogo.Client, error) {
	return miniogo.New(c.MinioEndpoint, &miniogo.Options{Secure: c.MinioSecure, Creds: credentials.NewStaticV4(c.MinioAccesskey, c.MinioSecretkey, "")})
}

type Storage struct {
	Client *miniogo.Client
	Bucket string
}

func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := s.Client.GetObject(ctx, s.Bucket, key, miniogo.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	if _, err = object.Stat(); err != nil {
		_ = object.Close()
		return nil, err
	}
	return object, nil
}
func (s *Storage) Put(ctx context.Context, key string, content io.Reader, size int64, contentType string) error {
	_, err := s.Client.PutObject(ctx, s.Bucket, key, content, size, miniogo.PutObjectOptions{ContentType: contentType})
	return err
}
func (s *Storage) Delete(ctx context.Context, key string) error {
	return s.Client.RemoveObject(ctx, s.Bucket, key, miniogo.RemoveObjectOptions{})
}
