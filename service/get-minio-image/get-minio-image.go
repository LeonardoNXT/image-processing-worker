package getMinioImage

import (
	"context"
	"fmt"
	"image-parsing-worker/config"

	"github.com/minio/minio-go/v7"
)

func Exec(objectKey string, minioClient *minio.Client, cfg *config.Config, ctx context.Context) *minio.Object {
	status, BucketErr := minioClient.BucketExists(ctx, cfg.MinioBucketName)

	if status != true {
		return nil
	}

	if BucketErr != nil {
		fmt.Println(BucketErr.Error())
	}

	object, err := minioClient.GetObject(ctx, cfg.MinioBucketName, objectKey, minio.GetObjectOptions{})

	if err != nil {
		fmt.Println(err.Error())
	}

	return object
}
