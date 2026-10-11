package usecases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image-processing-worker/application/ports"
	"image-processing-worker/domain"
	"image-processing-worker/domain/patterns"
	"io"
	"testing"
)

type fakeStorage struct {
	puts, deletes       []string
	failPut, failDelete bool
}

func (s *fakeStorage) Get(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}
func (s *fakeStorage) Put(_ context.Context, key string, _ io.Reader, _ int64, _ string) error {
	s.puts = append(s.puts, key)
	if s.failPut {
		return errors.New("storage unavailable")
	}
	return nil
}
func (s *fakeStorage) Delete(_ context.Context, key string) error {
	s.deletes = append(s.deletes, key)
	if s.failDelete {
		return errors.New("delete failed")
	}
	return nil
}

type fakeOutbox struct {
	row                                         *domain.Outbox
	failSave, commitDespiteError, failReconcile bool
	finds, saves                                int
}

func (o *fakeOutbox) Find(context.Context, string) (*domain.Outbox, error) {
	o.finds++
	if o.failReconcile && o.finds > 1 {
		return nil, errors.New("database unavailable")
	}
	return o.row, nil
}
func (o *fakeOutbox) Save(_ context.Context, row domain.Outbox) error {
	o.saves++
	if !o.failSave || o.commitDespiteError {
		o.row = &row
	}
	if o.failSave {
		return errors.New("save failed")
	}
	return nil
}

type fakeProcessor struct {
	calls int
	err   error
}

func (p *fakeProcessor) Process(context.Context, io.Reader, domain.Request) (*ports.Batch, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	return &ports.Batch{Close: func() error { return nil }, Artifacts: []ports.Artifact{
		{Content: bytes.NewReader([]byte("encoded")), Output: domain.Output{Name: "avatar", Format: "webp", ContentType: "image/webp", Width: 250, Height: 250, SizeBytes: 7, FrameCount: 1}},
	}}, nil
}
func request() domain.Request {
	return domain.Request{SchemaVersion: 1, EventID: "82c8e4ba-d96d-4076-9a73-109c908fd4bc", ImageID: "9a071332-a599-4559-b6b9-86d59e12d3e4", OwnerID: "owner", Service: "user", ObjectKey: "user/owner/profile-picture/image", OutputTopic: "user.processedImage", MaxSizeBytes: 5000000, PermittedFormats: []string{"png"}, OutputShape: []domain.OutputShape{{Name: "avatar", MaxWidth: 500, MaxHeight: 500}}}
}
func setup() (*ImagePipeline, *fakeStorage, *fakeStorage, *fakeOutbox, *fakeProcessor) {
	in, out, db, processor := &fakeStorage{}, &fakeStorage{}, &fakeOutbox{}, &fakeProcessor{}
	return &ImagePipeline{Incoming: in, Processed: out, Outbox: db, Processor: processor, ProcessedBucket: "processed", Limits: patterns.Config()}, in, out, db, processor
}
func TestSuccessAndDuplicate(t *testing.T) {
	p, in, out, db, proc := setup()
	r := request()
	if err := p.Run(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if len(out.puts) != 1 || out.puts[0] != r.ObjectKey+"/avatar/250x250.webp" || len(in.deletes) != 1 {
		t.Fatal("incorrect object lifecycle")
	}
	var result domain.Result
	if err := json.Unmarshal(db.row.Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.EventType != "IMAGE_PROCESSING_SUCCEEDED" || len(result.Outputs) != 1 || db.row.OutputTopic != r.OutputTopic {
		t.Fatalf("unexpected result: %+v", result)
	}
	if err := p.Run(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if proc.calls != 1 || db.saves != 1 {
		t.Fatal("duplicate processing")
	}
}
func TestDatabaseFailureCompensates(t *testing.T) {
	p, in, out, db, _ := setup()
	db.failSave = true
	if err := p.Run(context.Background(), request()); err == nil {
		t.Fatal("expected failure")
	}
	if len(out.deletes) != 1 || len(in.deletes) != 0 {
		t.Fatal("must rollback output and preserve input")
	}
}
func TestAmbiguousCommit(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(map[bool]string{false: "committed", true: "unknown"}[unknown], func(t *testing.T) {
			p, in, out, db, _ := setup()
			db.failSave = true
			db.commitDespiteError = true
			db.failReconcile = unknown
			err := p.Run(context.Background(), request())
			if (err != nil) != unknown {
				t.Fatalf("unexpected error %v", err)
			}
			if len(out.deletes) != 0 {
				t.Fatal("cannot delete possibly committed objects")
			}
			if unknown && len(in.deletes) != 0 {
				t.Fatal("must preserve source until reconciliation")
			}
		})
	}
}
func TestProcessingFailurePublishesAndCleansSource(t *testing.T) {
	p, in, out, db, proc := setup()
	proc.err = domain.Reject("INVALID_IMAGE", "DECODE", "invalid image")
	if err := p.Run(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	var result domain.Result
	_ = json.Unmarshal(db.row.Payload, &result)
	if result.EventType != "IMAGE_PROCESSING_FAILED" || result.Error == nil || len(result.Outputs) != 0 || len(out.puts) != 0 || len(in.deletes) != 1 {
		t.Fatal("incorrect rejection lifecycle")
	}
}
func TestInfrastructureFailureDoesNotPublish(t *testing.T) {
	p, in, _, db, proc := setup()
	proc.err = errors.New("engine unavailable")
	if err := p.Run(context.Background(), request()); err == nil {
		t.Fatal("expected error")
	}
	if db.saves != 0 || len(in.deletes) != 0 {
		t.Fatal("infrastructure failure must retain input without event")
	}
}
func TestUploadFailureCompensates(t *testing.T) {
	p, in, out, db, _ := setup()
	out.failPut = true
	if err := p.Run(context.Background(), request()); err == nil {
		t.Fatal("expected error")
	}
	if len(out.deletes) != 1 || db.saves != 0 || len(in.deletes) != 0 {
		t.Fatal("incorrect upload compensation")
	}
}
func TestCleanupRetryDoesNotRepublish(t *testing.T) {
	p, in, _, db, proc := setup()
	in.failDelete = true
	if err := p.Run(context.Background(), request()); err == nil {
		t.Fatal("expected error")
	}
	in.failDelete = false
	if err := p.Run(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if db.saves != 1 || proc.calls != 1 {
		t.Fatal("cleanup retry republished result")
	}
}

func TestRoutableSchemaFailureDoesNotRequireObjectKey(t *testing.T) {
	p, in, out, db, proc := setup()
	r := request()
	r.ObjectKey = ""
	failure := domain.Reject("INVALID_REQUEST", "VALIDATION", "request JSON does not match schema version 1")
	if err := p.Reject(context.Background(), r, failure); err != nil {
		t.Fatal(err)
	}
	if proc.calls != 0 || len(out.puts) != 0 || len(in.deletes) != 0 || db.saves != 1 {
		t.Fatal("routable schema failure used the image pipeline")
	}
	var result domain.Result
	if err := json.Unmarshal(db.row.Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.EventType != "IMAGE_PROCESSING_FAILED" || result.Error == nil || result.Error.Code != "INVALID_REQUEST" {
		t.Fatal("schema failure was not persisted")
	}
	if err := p.Reject(context.Background(), r, failure); err != nil {
		t.Fatal(err)
	}
	if db.saves != 1 || len(in.deletes) != 0 {
		t.Fatal("schema failure retry was not idempotent")
	}
}
