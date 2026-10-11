package worker

import (
	"context"
	"fmt"
	"github.com/twmb/franz-go/pkg/kgo"
	"image-processing-worker/application/usecases"
	"image-processing-worker/domain/patterns"
	"image-processing-worker/infrastructure/config"
	"image-processing-worker/infrastructure/dto"
	"image-processing-worker/infrastructure/messenger"
	"image-processing-worker/infrastructure/processing"
	storage "image-processing-worker/infrastructure/storage/minio"
	"image-processing-worker/infrastructure/storage/postgres"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func Main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := Run(ctx); err != nil && ctx.Err() == nil {
		slog.Error("worker stopped; uncommitted message will be retried", "error", err)
		os.Exit(1)
	}
}
func Run(ctx context.Context) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	startup, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	repo, err := postgres.New(startup, c)
	if err != nil {
		return err
	}
	defer repo.Pool.Close()
	if err = repo.Migrate(startup); err != nil {
		return fmt.Errorf("outbox migration: %w", err)
	}
	client, err := storage.Client(c)
	if err != nil {
		return err
	}
	for _, bucket := range []string{c.IncomingBucketName, c.ProcessedBucketName} {
		exists, err := client.BucketExists(startup, bucket)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("bucket %q must exist", bucket)
		}
	}
	limits := patterns.Config()
	processor, err := processing.New(c.ImageMagickBinary, limits)
	if err != nil {
		return err
	}
	pipeline := &usecases.ImagePipeline{Incoming: &storage.Storage{Client: client, Bucket: c.IncomingBucketName},
		Processed: &storage.Storage{Client: client, Bucket: c.ProcessedBucketName}, ProcessedBucket: c.ProcessedBucketName,
		Outbox: repo, Processor: processor, Limits: limits}
	kafka, err := messenger.Client(split(c.KafkaBrokers), split(c.KafkaTopics))
	if err != nil {
		return err
	}
	defer kafka.Close()
	slog.Info("image-processing-worker started")
	return messenger.Consumer(kafka, ctx, func(ctx context.Context, record *kgo.Record) error {
		request, failure, err := dto.DecodeIncomingPayload(record.Value)
		if err != nil {
			return fmt.Errorf("invalid JSON envelope: %w", err)
		}
		if err := request.ValidateRouting(); err != nil {
			return err
		}
		work, cancel := context.WithTimeout(ctx, 8*time.Minute)
		defer cancel()
		return repo.WithImageLock(work, request.ImageID, func() error {
			if failure != nil {
				return pipeline.Reject(work, request, failure)
			}
			return pipeline.Run(work, request)
		})
	})
}
func split(value string) []string {
	var values []string
	for _, v := range strings.Split(value, ",") {
		if v = strings.TrimSpace(v); v != "" {
			values = append(values, v)
		}
	}
	return values
}
