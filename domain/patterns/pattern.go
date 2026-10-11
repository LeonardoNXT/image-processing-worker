package patterns

import "strings"

type ImagePattern struct{ MaxPixels, MaxDecodedBytes int64 }
type AnimationPattern struct {
	MaxPixels, MaxDecodedBytes       int64
	ReferenceFrames, ReferencePixels int64
}
type Pattern struct {
	Image         ImagePattern
	Animation     AnimationPattern
	MaxInputBytes int64
	MaxOutputs    int
}

func Config() Pattern {
	return Pattern{
		Image:         ImagePattern{14_745_600, 58_982_400},
		Animation:     AnimationPattern{3_686_400, 14_745_600, 250, 2_073_600},
		MaxInputBytes: 64 << 20,
		MaxOutputs:    16,
	}
}
func NormalizeFormat(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "jpg" {
		return "jpeg"
	}
	return s
}
func InputSupported(s string) bool {
	switch NormalizeFormat(s) {
	case "jpeg", "png", "gif", "webp":
		return true
	}
	return false
}
func OutputSupported(s string) bool {
	switch NormalizeFormat(s) {
	case "jpeg", "png", "webp":
		return true
	}
	return false
}
