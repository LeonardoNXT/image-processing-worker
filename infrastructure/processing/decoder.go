package processing

import (
	"bytes"
	_ "golang.org/x/image/webp"
	"image"
	"image-processing-worker/domain"
	"image-processing-worker/domain/patterns"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type metadata struct {
	Format                string
	Width, Height, Frames int
	Animation             bool
}

func inspect(data []byte, fullAnimation bool, p patterns.Pattern) (metadata, error) {
	m := metadata{Frames: 1}
	switch {
	case len(data) >= 3 && bytes.Equal(data[:3], []byte{255, 216, 255}):
		m.Format = "jpeg"
	case len(data) >= 8 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")):
		m.Format = "png"
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		m.Format = "gif"
		m.Animation = true
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		m.Format = "webp"
	default:
		return m, domain.Reject("UNSUPPORTED_INPUT_FORMAT", "VALIDATION", "input is not JPEG, PNG, GIF or WebP")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return m, domain.Reject("INVALID_IMAGE", "DECODE", "invalid image header")
	}
	m.Width, m.Height = cfg.Width, cfg.Height
	if m.Format == "webp" && len(data) >= 21 && string(data[12:16]) == "VP8X" {
		m.Animation = data[20]&2 != 0
	}
	pixels := int64(m.Width) * int64(m.Height)
	limits := p.Image
	if m.Animation {
		limits = patterns.ImagePattern{MaxPixels: p.Animation.MaxPixels, MaxDecodedBytes: p.Animation.MaxDecodedBytes}
	}
	if m.Width <= 0 || m.Height <= 0 || pixels > limits.MaxPixels || pixels > limits.MaxDecodedBytes/4 {
		return m, domain.Reject("PIXEL_LIMIT_EXCEEDED", "VALIDATION", "decoded canvas exceeds worker pixel or RGBA budget")
	}
	if m.Animation {
		maxFrames := p.Animation.ReferenceFrames * p.Animation.ReferencePixels / pixels
		if !fullAnimation {
			maxFrames = -1
		}
		if m.Format == "gif" {
			m.Frames, err = gifFrames(data, maxFrames)
		} else {
			m.Frames, err = webpFrames(data, maxFrames)
		}
		if err != nil {
			return m, err
		}
		if m.Frames < 1 {
			return m, domain.Reject("INVALID_IMAGE", "DECODE", "animation has no frames")
		}
	}
	return m, nil
}
