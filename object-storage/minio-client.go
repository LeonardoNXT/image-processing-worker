package object_storage

import (
	"fmt"
	"image-parsing-worker/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func Client(config *config.Config) *minio.Client {
	cl, err := minio.New(config.MinioEndpoint, &minio.Options{
		Secure: false,
		Creds:  credentials.NewStaticV4(config.MinioAccesskey, config.MinioSecretkey, ""),
	})

	if err != nil {
		fmt.Println(err.Error())
	}

	return cl

}
