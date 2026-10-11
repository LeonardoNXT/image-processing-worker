package domain

import (
	"fmt"
	"github.com/google/uuid"
	"image-processing-worker/domain/patterns"
	"regexp"
	"strings"
	"time"
)

type Request struct {
	SchemaVersion    int           `json:"schemaVersion"`
	EventID          string        `json:"eventId"`
	ImageID          string        `json:"imageId"`
	OwnerID          string        `json:"ownerId"`
	ObjectKey        string        `json:"objectKey"`
	Service          string        `json:"service"`
	MaxSizeBytes     int64         `json:"maxSizeBytes"`
	OutputShape      []OutputShape `json:"outputShape"`
	OutputTopic      string        `json:"outputTopic"`
	PermittedFormats []string      `json:"permittedFormats"`
}
type OutputShape struct {
	Name      string  `json:"name"`
	MaxWidth  int     `json:"maxWidth"`
	MaxHeight int     `json:"maxHeight"`
	Format    *string `json:"format"`
	Quality   *int    `json:"quality"`
}

func (s OutputShape) Encoding() string {
	if s.Format == nil {
		return "webp"
	}
	return patterns.NormalizeFormat(*s.Format)
}
func (s OutputShape) EncodingQuality() int {
	if s.Quality == nil {
		return 80
	}
	return *s.Quality
}

type ProcessingError struct {
	Code       string  `json:"code"`
	Stage      string  `json:"stage"`
	Message    string  `json:"message"`
	OutputName *string `json:"outputName"`
}

func (e *ProcessingError) Error() string { return e.Code + ": " + e.Message }
func Reject(code, stage, message string) *ProcessingError {
	return &ProcessingError{Code: code, Stage: stage, Message: message}
}

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var topicPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,249}$`)

func (r Request) ValidateRouting() error {
	for _, id := range []string{r.EventID, r.ImageID} {
		if _, err := uuid.Parse(id); err != nil {
			return fmt.Errorf("eventId and imageId must be UUIDs")
		}
	}
	if !topicPattern.MatchString(r.OutputTopic) || r.OutputTopic == "." || r.OutputTopic == ".." {
		return fmt.Errorf("invalid outputTopic")
	}
	return nil
}
func (r Request) Validate(p patterns.Pattern) *ProcessingError {
	bad := func(message string) *ProcessingError { return Reject("INVALID_REQUEST", "VALIDATION", message) }
	if r.SchemaVersion != 1 {
		return bad("schemaVersion must be 1")
	}
	if strings.TrimSpace(r.Service) == "" || strings.TrimSpace(r.OwnerID) == "" {
		return bad("service and ownerId are required")
	}
	if !r.HasValidObjectKey() {
		return bad("objectKey is invalid")
	}
	if r.MaxSizeBytes <= 0 {
		return bad("maxSizeBytes must be positive")
	}
	if len(r.PermittedFormats) == 0 {
		return bad("permittedFormats must not be empty")
	}
	if len(r.OutputShape) == 0 || len(r.OutputShape) > p.MaxOutputs {
		return bad(fmt.Sprintf("outputShape must contain 1 to %d items", p.MaxOutputs))
	}
	names := map[string]bool{}
	for _, s := range r.OutputShape {
		if !namePattern.MatchString(s.Name) || names[s.Name] {
			return bad("output names must be unique safe path segments (1 to 64 characters)")
		}
		names[s.Name] = true
		if s.MaxWidth <= 0 || s.MaxHeight <= 0 || int64(s.MaxWidth) > p.Image.MaxPixels || int64(s.MaxHeight) > p.Image.MaxPixels || int64(s.MaxWidth)*int64(s.MaxHeight) > p.Image.MaxPixels {
			return bad("output dimensions exceed worker limits or are not positive")
		}
		if !patterns.OutputSupported(s.Encoding()) {
			e := Reject("UNSUPPORTED_OUTPUT_FORMAT", "VALIDATION", "outputs support only JPEG/JPG, PNG and WebP")
			e.OutputName = &s.Name
			return e
		}
		if s.Encoding() != "png" && (s.EncodingQuality() < 1 || s.EncodingQuality() > 100) {
			return bad("quality must be between 1 and 100")
		}
	}
	return nil
}

func (r Request) HasValidObjectKey() bool {
	if r.ObjectKey == "" || strings.HasPrefix(r.ObjectKey, "/") || strings.HasSuffix(r.ObjectKey, "/") || strings.ContainsAny(r.ObjectKey, "\\\x00\r\n") {
		return false
	}
	for _, segment := range strings.Split(r.ObjectKey, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}
func (r Request) Allows(format string) bool {
	if !patterns.InputSupported(format) {
		return false
	}
	for _, f := range r.PermittedFormats {
		if patterns.NormalizeFormat(f) == patterns.NormalizeFormat(format) {
			return true
		}
	}
	return false
}
func (r Request) NeedsAnimation() bool {
	for _, s := range r.OutputShape {
		if s.Encoding() == "webp" {
			return true
		}
	}
	return false
}

type Output struct {
	Name        string `json:"name"`
	Bucket      string `json:"bucket"`
	ObjectKey   string `json:"objectKey"`
	Format      string `json:"format"`
	ContentType string `json:"contentType"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	SizeBytes   int64  `json:"sizeBytes"`
	Animated    bool   `json:"animated"`
	FrameCount  int    `json:"frameCount"`
}
type Result struct {
	SchemaVersion   int              `json:"schemaVersion"`
	EventID         string           `json:"eventId"`
	RequestEventID  string           `json:"requestEventId"`
	EventType       string           `json:"eventType"`
	OccurredAt      time.Time        `json:"occurredAt"`
	ImageID         string           `json:"imageId"`
	OwnerID         string           `json:"ownerId"`
	Service         string           `json:"service"`
	SourceObjectKey string           `json:"sourceObjectKey"`
	Outputs         []Output         `json:"outputs"`
	Error           *ProcessingError `json:"error"`
}
