package bootstrap

import (
	"context"
	"image-parsing-worker/config"
	"image-parsing-worker/messenger"
	"image-parsing-worker/object-storage"
	"image-parsing-worker/service"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/twmb/franz-go/pkg/kgo"
)

func Bootstrap() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer stop()

	cfg, err := config.Load()

	if err != nil {
		panic(err.Error())
	}

	kfkClient, kfkErr := messenger.Client(strings.Split(cfg.KafkaBrokers, ","), strings.Split(cfg.KafkaTopics, ","))

	if kfkErr != nil {
		panic(kfkErr.Error())
	}

	minioClient := object_storage.Client(cfg)

	messenger.Consumer(kfkClient, ctx, func(record *kgo.Record) {
		service.Start(record, ctx, minioClient)
	})
}
