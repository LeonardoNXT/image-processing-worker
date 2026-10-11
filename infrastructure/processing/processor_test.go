package processing

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image-processing-worker/application/ports"
	"image-processing-worker/domain"
	"image-processing-worker/domain/patterns"
	"image/color"
	"image/gif"
	"image/png"
	"io"
	"os/exec"
	"testing"
)

func TestCenterCrop(t *testing.T) {
	for _, tc := range []struct{ w, h, maxW, maxH, cropW, cropH, outW, outH int }{
		{1200, 1100, 1200, 1500, 880, 1100, 880, 1100},
		{1920, 1080, 500, 500, 1080, 1080, 500, 500},
		{250, 250, 500, 500, 250, 250, 250, 250},
	} {
		r := ports.Reference{Width: tc.maxW, Height: tc.maxH}
		c := CenterCrop(tc.w, tc.h, r)
		w, h := outputSize(c, r)
		if c.Width != tc.cropW || c.Height != tc.cropH || w != tc.outW || h != tc.outH {
			t.Fatalf("unexpected crop %+v -> %dx%d", c, w, h)
		}
	}
}
func pngInput(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func gifInput(t *testing.T) []byte {
	return gifInputFrames(t, 2)
}
func gifInputFrames(t *testing.T, count int) []byte {
	t.Helper()
	pal := color.Palette{color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}}
	a := image.NewPaletted(image.Rect(0, 0, 16, 16), pal)
	b := image.NewPaletted(image.Rect(0, 0, 16, 16), pal)
	for i := range b.Pix {
		b.Pix[i] = 1
	}
	var buf bytes.Buffer
	animation := &gif.GIF{LoopCount: 0}
	for i := 0; i < count; i++ {
		animation.Image = append(animation.Image, []*image.Paletted{a, b}[i%2])
		animation.Delay = append(animation.Delay, 10*(1+i%2))
	}
	if err := gif.EncodeAll(&buf, animation); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestManyFramesToStatic(t *testing.T) {
	p := engine(t)
	animated, err := p.Process(context.Background(), bytes.NewReader(gifInputFrames(t, 20)), testRequest("webp"))
	if err != nil {
		t.Fatal(err)
	}
	defer animated.Close()
	webp, err := io.ReadAll(animated.Artifacts[0].Content)
	if err != nil {
		t.Fatal(err)
	}
	p.Limits.Animation.ReferenceFrames = 1
	p.Limits.Animation.ReferencePixels = 256
	for name, data := range map[string][]byte{"gif": gifInputFrames(t, 20), "webp": webp} {
		t.Run(name, func(t *testing.T) {
			b, err := p.Process(context.Background(), bytes.NewReader(data), testRequest("png"))
			if err != nil {
				t.Fatal(err)
			}
			defer b.Close()
			decoded, err := png.Decode(b.Artifacts[0].Content)
			if err != nil {
				t.Fatal(err)
			}
			r, _, blue, _ := decoded.At(8, 8).RGBA()
			if r < 50000 || blue > 10000 {
				t.Fatal("wrong first frame")
			}
		})
	}
}
func testRequest(format string) domain.Request {
	r := domain.Request{SchemaVersion: 1, OwnerID: "owner", Service: "test", ObjectKey: "test/image", MaxSizeBytes: 1 << 20, PermittedFormats: []string{"png", "gif", "webp", "jpg"}, OutputShape: []domain.OutputShape{{Name: "avatar", MaxWidth: 32, MaxHeight: 32}}}
	if format != "" {
		r.OutputShape[0].Format = &format
	}
	return r
}
func engine(t *testing.T) *Processor {
	t.Helper()
	if _, err := exec.LookPath("magick"); err != nil {
		t.Skip("ImageMagick 7 unavailable")
	}
	p, err := New("magick", patterns.Config())
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestRealStaticEncodes(t *testing.T) {
	p := engine(t)
	for _, format := range []string{"jpg", "jpeg", "png", "webp", ""} {
		t.Run(format, func(t *testing.T) {
			batch, err := p.Process(context.Background(), bytes.NewReader(pngInput(t, 16, 16)), testRequest(format))
			if err != nil {
				t.Fatal(err)
			}
			defer batch.Close()
			b, err := io.ReadAll(batch.Artifacts[0].Content)
			if err != nil {
				t.Fatal(err)
			}
			cfg, detected, err := image.DecodeConfig(bytes.NewReader(b))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Width != 16 || cfg.Height != 16 || detected != testRequest(format).OutputShape[0].Encoding() {
				t.Fatalf("unexpected %s %dx%d", detected, cfg.Width, cfg.Height)
			}
		})
	}
}
func TestRealAnimatedWebPRoundTrip(t *testing.T) {
	p := engine(t)
	batch, err := p.Process(context.Background(), bytes.NewReader(gifInput(t)), testRequest("webp"))
	if err != nil {
		t.Fatal(err)
	}
	defer batch.Close()
	encoded, err := io.ReadAll(batch.Artifacts[0].Content)
	if err != nil {
		t.Fatal(err)
	}
	n, err := webpFrames(encoded, 100)
	if err != nil || n != 2 {
		t.Fatalf("frames=%d err=%v", n, err)
	}
	for _, format := range []string{"webp", "jpeg", "png"} {
		t.Run(format, func(t *testing.T) {
			b, err := p.Process(context.Background(), bytes.NewReader(encoded), testRequest(format))
			if err != nil {
				t.Fatal(err)
			}
			defer b.Close()
			if format == "webp" && !b.Artifacts[0].Output.Animated {
				t.Fatal("animation lost")
			}
			if format != "webp" && b.Artifacts[0].Output.FrameCount != 1 {
				t.Fatal("expected first frame only")
			}
		})
	}
}
func TestStaticOutputIgnoresFrameBudget(t *testing.T) {
	p := engine(t)
	p.Limits.Animation.ReferenceFrames = 1
	p.Limits.Animation.ReferencePixels = 256
	data := gifInput(t)
	_, err := p.Process(context.Background(), bytes.NewReader(data), testRequest("webp"))
	var rejection *domain.ProcessingError
	if !errors.As(err, &rejection) || rejection.Code != "FRAME_LIMIT_EXCEEDED" {
		t.Fatalf("expected frame rejection: %v", err)
	}
	batch, err := p.Process(context.Background(), bytes.NewReader(data), testRequest("jpeg"))
	if err != nil {
		t.Fatal(err)
	}
	defer batch.Close()
}
func TestHeaderLimitsAndServiceRestrictions(t *testing.T) {
	p := engine(t)
	r := testRequest("webp")
	r.PermittedFormats = []string{"jpeg"}
	_, err := p.Process(context.Background(), bytes.NewReader(pngInput(t, 16, 16)), r)
	var rejection *domain.ProcessingError
	if !errors.As(err, &rejection) || rejection.Code != "INPUT_FORMAT_NOT_PERMITTED" {
		t.Fatalf("unexpected %v", err)
	}
	r = testRequest("webp")
	r.MaxSizeBytes = 8
	_, err = p.Process(context.Background(), bytes.NewReader(pngInput(t, 16, 16)), r)
	if !errors.As(err, &rejection) || rejection.Code != "INPUT_TOO_LARGE" {
		t.Fatalf("unexpected %v", err)
	}
	data := gifInput(t)
	binary.LittleEndian.PutUint16(data[6:8], 65535)
	binary.LittleEndian.PutUint16(data[8:10], 65535)
	_, err = inspect(data, true, p.Limits)
	if !errors.As(err, &rejection) || rejection.Code != "PIXEL_LIMIT_EXCEEDED" {
		t.Fatalf("unexpected %v", err)
	}
}
