package service

import (
	"context"
	"image-parsing-worker/config"
	"image-parsing-worker/service/get-minio-image"
	"image-parsing-worker/service/image-decoder"
	"image-parsing-worker/service/incoming-payload-normalizer"

	"github.com/minio/minio-go/v7"
	"github.com/twmb/franz-go/pkg/kgo"
)

func Start(record *kgo.Record, cfg *config.Config, ctx context.Context, minioClient *minio.Client) {
	incomingPayload := incomingPayloadNormalizer.Exec(record.Value)
	object := getMinioImage.Exec(incomingPayload.ObjectKey, minioClient, cfg, ctx)

	src, format, decoderError := imageDecoder.Exec(object, incomingPayload.PermittedFormats)

	if decoderError != nil {

	}

}
