package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	KafkaBrokers, KafkaTopics                     string
	MinioEndpoint, MinioAccesskey, MinioSecretkey string
	IncomingBucketName, ProcessedBucketName       string
	MinioSecure                                   bool
	PostgresURL, PostgresUser, PostgresPassword   string
	ImageMagickBinary                             string
}

func Load() (*Config, error) {
	c := &Config{
		KafkaBrokers: os.Getenv("KAFKA_BROKERS"), KafkaTopics: os.Getenv("KAFKA_TOPICS"),
		MinioEndpoint: os.Getenv("MINIO_ENDPOINT"), MinioAccesskey: os.Getenv("MINIO_ACCESS_KEY"), MinioSecretkey: os.Getenv("MINIO_SECRET_KEY"),
		IncomingBucketName: os.Getenv("MINIO_INCOMING_BUCKET_NAME"), ProcessedBucketName: os.Getenv("MINIO_PROCESSED_BUCKET_NAME"),
		PostgresURL: os.Getenv("POSTGRES_URL"), PostgresUser: os.Getenv("POSTGRES_USER"), PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		ImageMagickBinary: os.Getenv("IMAGEMAGICK_BINARY"),
	}
	if c.ImageMagickBinary == "" {
		c.ImageMagickBinary = "magick"
	}
	if value := os.Getenv("MINIO_SECURE"); value != "" {
		var err error
		c.MinioSecure, err = strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("MINIO_SECURE: %w", err)
		}
	}
	for _, entry := range [][2]string{{"KAFKA_BROKERS", c.KafkaBrokers}, {"KAFKA_TOPICS", c.KafkaTopics}, {"MINIO_ENDPOINT", c.MinioEndpoint}, {"MINIO_ACCESS_KEY", c.MinioAccesskey}, {"MINIO_SECRET_KEY", c.MinioSecretkey}, {"MINIO_INCOMING_BUCKET_NAME", c.IncomingBucketName}, {"MINIO_PROCESSED_BUCKET_NAME", c.ProcessedBucketName}, {"POSTGRES_URL", c.PostgresURL}} {
		if entry[1] == "" {
			return nil, fmt.Errorf("%s is required", entry[0])
		}
	}
	if c.IncomingBucketName == c.ProcessedBucketName {
		return nil, fmt.Errorf("incoming and processed buckets must differ")
	}
	return c, nil
}
