package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"image-processing-worker/application/ports"
	"image-processing-worker/domain"
	"image-processing-worker/infrastructure/config"
	"image-processing-worker/observability"
	"time"
)

//go:embed 001_outbox.sql
var migration string

type Repository struct{ Pool *pgxpool.Pool }

func New(ctx context.Context, c *config.Config) (*Repository, error) {
	pc, err := pgxpool.ParseConfig(c.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL configuration")
	}
	if c.PostgresUser != "" {
		pc.ConnConfig.User = c.PostgresUser
	}
	if c.PostgresPassword != "" {
		pc.ConnConfig.Password = c.PostgresPassword
	}
	pc.ConnConfig.ConnectTimeout = 5 * time.Second
	pc.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Repository{Pool: pool}, nil
}
func (r *Repository) Migrate(ctx context.Context) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724036241)"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, migration); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) Find(ctx context.Context, requestID string) (*domain.Outbox, error) {
	var o domain.Outbox
	err := r.Pool.QueryRow(ctx, `SELECT id::text, request_event_id::text, aggregate_id::text,
        event_type, aggregate_type, reason, payload, output_topic, occurred_at,
        retry_limit, retry_count, status, processing_started_at, last_attempt_at,
        published_at, created_at, updated_at FROM outbox WHERE request_event_id=$1`, requestID).Scan(
		&o.ID, &o.RequestEventID, &o.AggregateID, &o.EventType, &o.AggregateType, &o.Reason, &o.Payload, &o.OutputTopic, &o.OccurredAt,
		&o.RetryLimit, &o.RetryCount, &o.Status, &o.ProcessingStartedAt, &o.LastAttemptAt, &o.PublishedAt, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}
func (r *Repository) Save(ctx context.Context, o domain.Outbox) error {
	_, err := r.Pool.Exec(ctx, `INSERT INTO outbox
        (id,request_event_id,aggregate_id,event_type,aggregate_type,reason,payload,output_topic,
         occurred_at,retry_limit,retry_count,status,processing_started_at,last_attempt_at,published_at,created_at,updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		o.ID, o.RequestEventID, o.AggregateID, o.EventType, o.AggregateType, o.Reason, string(o.Payload), o.OutputTopic,
		o.OccurredAt, o.RetryLimit, o.RetryCount, o.Status, o.ProcessingStartedAt, o.LastAttemptAt, o.PublishedAt, o.CreatedAt, o.UpdatedAt)
	var serverError *pgconn.PgError
	if err != nil && (pgconn.SafeToRetry(err) || errors.As(err, &serverError)) {
		if serverError == nil || serverError.Code != "23505" {
			return errors.Join(ports.ErrOutboxNotSaved, err)
		}
	}
	return err
}

func (r *Repository) WithImageLock(ctx context.Context, imageID string, fn func() error) error {
	c, err := r.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer c.Release()
	if _, err = c.Exec(ctx, "SELECT pg_advisory_lock(hashtextextended($1, 0))", imageID); err != nil {
		_ = c.Conn().Close(context.Background())
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := c.Exec(cleanup, "SELECT pg_advisory_unlock(hashtextextended($1, 0))", imageID); err != nil {
			observability.CleanupFailure("postgres_image_lock", err)
			_ = c.Conn().Close(cleanup)
		}
	}()
	return fn()
}
