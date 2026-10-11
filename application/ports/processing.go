package ports

import (
	"context"
	"errors"
	"image-processing-worker/domain"
	"io"
)

var ErrOutboxNotSaved = errors.New("outbox was not saved")

type Artifact struct {
	Output  domain.Output
	Content io.ReadSeeker
}
type Batch struct {
	Artifacts []Artifact
	Close     func() error
}
type Processor interface {
	Process(context.Context, io.Reader, domain.Request) (*Batch, error)
}

type Outbox interface {
	Find(context.Context, string) (*domain.Outbox, error)
	Save(context.Context, domain.Outbox) error
}
