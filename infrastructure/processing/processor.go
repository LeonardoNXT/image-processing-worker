package processing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image-processing-worker/application/ports"
	"image-processing-worker/domain"
	"image-processing-worker/domain/patterns"
	"image-processing-worker/observability"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Processor struct {
	Binary string
	Limits patterns.Pattern
}

func New(binary string, limits patterns.Pattern) (*Processor, error) {
	path, err := exec.LookPath(binary)
	if err != nil {
		return nil, fmt.Errorf("ImageMagick 7 with WebP support is required: %w", err)
	}
	return &Processor{Binary: path, Limits: limits}, nil
}
func (p *Processor) Process(ctx context.Context, source io.Reader, r domain.Request) (_ *ports.Batch, retErr error) {
	if e := r.Validate(p.Limits); e != nil {
		return nil, e
	}
	limit := min(r.MaxSizeBytes, p.Limits.MaxInputBytes)
	data, err := io.ReadAll(io.LimitReader(source, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, domain.Reject("INPUT_TOO_LARGE", "VALIDATION", "input file exceeds the service or worker byte limit")
	}
	m, err := inspect(data, r.NeedsAnimation(), p.Limits)
	if err != nil {
		return nil, err
	}
	if !r.Allows(m.Format) {
		return nil, domain.Reject("INPUT_FORMAT_NOT_PERMITTED", "VALIDATION", "detected input format is not permitted by the service")
	}
	dir, err := os.MkdirTemp("", "image-processing-*")
	if err != nil {
		return nil, err
	}
	files := []*os.File{}
	cleanup := func() error {
		var errs []error
		for _, f := range files {
			errs = append(errs, f.Close())
		}
		errs = append(errs, os.RemoveAll(dir))
		return errors.Join(errs...)
	}
	defer func() {
		if retErr != nil {
			observability.CleanupFailure("temporary_processing_directory", cleanup())
		}
	}()
	input := filepath.Join(dir, "source."+m.Format)
	if err = os.WriteFile(input, data, 0600); err != nil {
		return nil, err
	}
	data = nil
	batch := &ports.Batch{Close: cleanup}
	for i, shape := range r.OutputShape {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		crop := CenterCrop(m.Width, m.Height, ports.Reference{Width: shape.MaxWidth, Height: shape.MaxHeight})
		w, h := outputSize(crop, ports.Reference{Width: shape.MaxWidth, Height: shape.MaxHeight})
		format := shape.Encoding()
		output := filepath.Join(dir, fmt.Sprintf("output-%d.%s", i, format))
		animate := m.Animation && format == "webp" && m.Frames > 1
		sourceArg := m.Format + ":" + input
		if !animate {
			sourceArg += "[0]"
		}
		args := []string{"-limit", "memory", "256MiB", "-limit", "map", "0", "-limit", "disk", "1GiB", "-limit", "thread", "1",
			"-limit", "list-length", fmt.Sprint(max(m.Frames, 1) + 1), sourceArg}
		if m.Animation {
			args = append(args, "-coalesce")
		}
		args = append(args, "-crop", fmt.Sprintf("%dx%d+%d+%d", crop.Width, crop.Height, crop.X, crop.Y), "+repage",
			"-resize", fmt.Sprintf("%dx%d!", w, h), "-strip")
		if format == "jpeg" {
			args = append(args, "-background", "white", "-alpha", "remove", "-alpha", "off")
		}
		if format != "png" {
			args = append(args, "-quality", fmt.Sprint(shape.EncodingQuality()))
		}
		args = append(args, format+":"+output)
		commandCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		cmd := exec.CommandContext(commandCtx, p.Binary, args...)
		cmd.Env = append(os.Environ(), "MAGICK_TEMPORARY_PATH="+dir)
		stderr := &limitedLog{}
		cmd.Stderr = stderr
		err = cmd.Run()
		commandErr := commandCtx.Err()
		cancel()
		if err != nil {
			if commandErr != nil {
				return nil, fmt.Errorf("image engine timeout/cancellation: %w", commandErr)
			}
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				return nil, fmt.Errorf("run image engine: %w", err)
			}
			detail := strings.ToLower(stderr.String())
			for _, term := range []string{"cache resources exhausted", "memory allocation failed", "no space left", "unable to open", "no encode delegate", "no decode delegate", "not authorized", "security policy"} {
				if strings.Contains(detail, term) {
					return nil, fmt.Errorf("image engine unavailable: %s", stderr.String())
				}
			}
			e := domain.Reject("IMAGE_PROCESSING_FAILED", "ENCODE", "image could not be decoded or encoded")
			e.OutputName = &shape.Name
			return nil, e
		}
		f, err := os.Open(output)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
		stat, err := f.Stat()
		if err != nil {
			return nil, err
		}
		actualWidth, actualHeight, actualFrames, err := encodedMetadata(f, format)
		if err != nil {
			return nil, fmt.Errorf("inspect encoded output: %w", err)
		}
		batch.Artifacts = append(batch.Artifacts, ports.Artifact{Content: f, Output: domain.Output{
			Name: shape.Name, Format: format, ContentType: "image/" + format, Width: actualWidth, Height: actualHeight, SizeBytes: stat.Size(), Animated: actualFrames > 1, FrameCount: actualFrames}})
	}
	return batch, nil
}

type limitedLog struct{ bytes.Buffer }

func (b *limitedLog) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len() < 8192 {
		_, _ = b.Buffer.Write(p[:min(len(p), 8192-b.Len())])
	}
	return n, nil
}
