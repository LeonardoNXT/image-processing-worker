package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"image-processing-worker/application/ports"
	"image-processing-worker/domain"
	"image-processing-worker/domain/patterns"
	"image-processing-worker/observability"
	"time"
)

type ImagePipeline struct {
	Incoming, Processed ports.ObjectStorage
	Outbox              ports.Outbox
	Processor           ports.Processor
	ProcessedBucket     string
	Limits              patterns.Pattern
}

func (p *ImagePipeline) Run(ctx context.Context, r domain.Request) error {
	return p.run(ctx, r, nil)
}

func (p *ImagePipeline) Reject(ctx context.Context, r domain.Request, failure *domain.ProcessingError) error {
	if failure == nil {
		return errors.New("processing failure is required")
	}
	return p.run(ctx, r, failure)
}

func (p *ImagePipeline) run(ctx context.Context, r domain.Request, failure *domain.ProcessingError) error {
	if err := r.ValidateRouting(); err != nil {
		return err
	}
	previous, err := p.Outbox.Find(ctx, r.EventID)
	if err != nil {
		return fmt.Errorf("lookup outbox: %w", err)
	}
	if previous != nil {
		var saved domain.Result
		if err := json.Unmarshal(previous.Payload, &saved); err != nil {
			return err
		}
		if saved.SourceObjectKey != r.ObjectKey || saved.ImageID != r.ImageID || previous.OutputTopic != r.OutputTopic {
			return fmt.Errorf("request event ID reused with different identity")
		}
		if !r.HasValidObjectKey() {
			return nil
		}
		return p.deleteIncoming(ctx, r.ObjectKey)
	}
	if failure == nil {
		failure = r.Validate(p.Limits)
	}
	var batch *ports.Batch
	if failure == nil {
		source, err := p.Incoming.Get(ctx, r.ObjectKey)
		if err != nil {
			return fmt.Errorf("get incoming: %w", err)
		}
		batch, err = p.Processor.Process(ctx, source, r)
		closeErr := source.Close()
		observability.CleanupFailure("incoming_object", closeErr)
		if err != nil {
			if !errors.As(err, &failure) {
				return fmt.Errorf("process: %w", err)
			}
		} else if closeErr != nil {
			if batch != nil {
				observability.CleanupFailure("processing_batch", batch.Close())
			}
			return closeErr
		}
	}
	if batch != nil {
		defer func() {
			observability.CleanupFailure("processing_batch", batch.Close())
		}()
	}
	now := time.Now().UTC()
	result := domain.Result{SchemaVersion: 1, EventID: uuid.NewString(), RequestEventID: r.EventID,
		EventType: "IMAGE_PROCESSING_SUCCEEDED", OccurredAt: now, ImageID: r.ImageID, OwnerID: r.OwnerID,
		Service: r.Service, SourceObjectKey: r.ObjectKey, Outputs: []domain.Output{}, Error: failure}
	written := []string{}
	rollback := func(cause error) error {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		for _, key := range written {
			if err := p.Processed.Delete(cleanupCtx, key); err != nil {
				observability.CleanupFailure("processed_object_rollback", err)
				cause = errors.Join(cause, fmt.Errorf("rollback %s: %w", key, err))
			}
		}
		return cause
	}
	if failure != nil {
		result.EventType = "IMAGE_PROCESSING_FAILED"
	} else {
		if batch == nil {
			return errors.New("processor returned no batch")
		}
		for _, artifact := range batch.Artifacts {
			o := artifact.Output
			o.Bucket = p.ProcessedBucket
			o.ObjectKey = fmt.Sprintf("%s/%s/%dx%d.%s", r.ObjectKey, o.Name, o.Width, o.Height, o.Format)
			written = append(written, o.ObjectKey)
			if err := p.Processed.Put(ctx, o.ObjectKey, artifact.Content, o.SizeBytes, o.ContentType); err != nil {
				return rollback(fmt.Errorf("put processed: %w", err))
			}
			result.Outputs = append(result.Outputs, o)
		}
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return rollback(err)
	}
	row := domain.Outbox{ID: result.EventID, RequestEventID: r.EventID, AggregateID: r.ImageID, AggregateType: "IMAGE",
		EventType: result.EventType, Payload: payload, OutputTopic: r.OutputTopic, OccurredAt: now,
		RetryLimit: 10, Status: "PENDING", CreatedAt: now, UpdatedAt: now}
	if failure != nil {
		row.Reason = &failure.Code
	}
	if err = p.Outbox.Save(ctx, row); err != nil {
		if errors.Is(err, ports.ErrOutboxNotSaved) {
			return rollback(fmt.Errorf("save outbox: %w", err))
		}
		checkCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		persisted, lookupErr := p.Outbox.Find(checkCtx, r.EventID)
		cancel()
		if lookupErr != nil {
			return errors.Join(fmt.Errorf("outbox outcome unknown; objects retained: %w", err), lookupErr)
		}
		if persisted == nil {
			return rollback(fmt.Errorf("save outbox: %w", err))
		}
		if persisted.ID != row.ID {
			return fmt.Errorf("concurrent result for request; objects retained")
		}
	}
	if !r.HasValidObjectKey() {
		return nil
	}
	return p.deleteIncoming(ctx, r.ObjectKey)
}

func (p *ImagePipeline) deleteIncoming(ctx context.Context, key string) error {
	err := p.Incoming.Delete(ctx, key)
	observability.CleanupFailure("incoming_object", err)
	return err
}
