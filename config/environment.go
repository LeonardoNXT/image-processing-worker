package config

import (
	"fmt"
	"os"
)

type Config struct {
	KafkaBrokers     string
	KafkaTopics      string
	MinioEndpoint    string
	MinioAccesskey   string
	MinioSecretkey   string
	MinioBucketName  string
	PostgresURL      string
	PostgresUser     string
	PostgresPassword string
}

func Load() (*Config, error) {

	cfg := &Config{
		KafkaBrokers:     os.Getenv("KAFKA_BROKERS"),
		KafkaTopics:      os.Getenv("KAFKA_TOPICS"),
		MinioEndpoint:    os.Getenv("MINIO_ENDPOINT"),
		MinioBucketName:  os.Getenv("MINIO_BUCKET_NAME"),
		MinioAccesskey:   os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretkey:   os.Getenv("MINIO_SECRET_KEY"),
		PostgresURL:      os.Getenv("POSTGRES_URL"),
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
	}

	if (cfg.KafkaBrokers) == "" {
		return nil, fmt.Errorf("Kafka broker is required")
	}

	if (cfg.KafkaTopics) == "" {
		return nil, fmt.Errorf("Kafka topics is required")
	}

	if (cfg.MinioEndpoint) == "" {
		return nil, fmt.Errorf("Minio endpoint is required")
	}

	if (cfg.MinioBucketName) == "" {
		return nil, fmt.Errorf("Minio bucket is required")
	}

	if (cfg.MinioAccesskey) == "" {
		return nil, fmt.Errorf("Minio access key is required")
	}

	if (cfg.MinioSecretkey) == "" {
		return nil, fmt.Errorf("Minio secret key is required")
	}

	if (cfg.PostgresURL) == "" {
		return nil, fmt.Errorf("Postgres URL is required")
	}

	if (cfg.PostgresUser) == "" {
		return nil, fmt.Errorf("Postgres user is required")
	}

	return cfg, nil
}
