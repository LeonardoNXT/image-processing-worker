package processing

import (
	"encoding/binary"
	"image-processing-worker/domain"
)

func invalidAnimation() error {
	return domain.Reject("INVALID_IMAGE", "DECODE", "invalid animation structure")
}
func frameLimit() error {
	return domain.Reject("FRAME_LIMIT_EXCEEDED", "VALIDATION", "animation exceeds the pixel/frame budget")
}
func webpFrames(b []byte, limit int64) (int, error) {
	if len(b) < 12 || int64(binary.LittleEndian.Uint32(b[4:8]))+8 != int64(len(b)) {
		return 0, invalidAnimation()
	}
	n := 0
	for pos := 12; pos < len(b); {
		if len(b)-pos < 8 {
			return 0, invalidAnimation()
		}
		size := int64(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
		end := int64(pos) + 8 + size
		if end > int64(len(b)) {
			return 0, invalidAnimation()
		}
		if string(b[pos:pos+4]) == "ANMF" {
			if size < 16 {
				return 0, invalidAnimation()
			}
			if len(b) < 30 || string(b[12:16]) != "VP8X" {
				return 0, invalidAnimation()
			}
			header := b[pos+8 : pos+24]
			x, y := uint24(header[:3])*2, uint24(header[3:6])*2
			w, h := uint24(header[6:9])+1, uint24(header[9:12])+1
			if x+w > uint24(b[24:27])+1 || y+h > uint24(b[27:30])+1 {
				return 0, invalidAnimation()
			}
			n++
			if limit < 0 {
				return n, nil
			}
			if int64(n) > limit {
				return n, frameLimit()
			}
		}
		pos = int(end + size%2)
		if pos > len(b) {
			return 0, invalidAnimation()
		}
	}
	return n, nil
}
func gifFrames(b []byte, limit int64) (int, error) {
	if len(b) < 13 {
		return 0, invalidAnimation()
	}
	pos := 13
	if b[10]&128 != 0 {
		pos += 3 * (1 << ((b[10] & 7) + 1))
	}
	skipBlocks := func() bool {
		for pos < len(b) {
			size := int(b[pos])
			pos++
			if size == 0 {
				return true
			}
			pos += size
		}
		return false
	}
	n := 0
	for pos < len(b) {
		marker := b[pos]
		pos++
		switch marker {
		case 0x3b:
			return n, nil
		case 0x21:
			if pos >= len(b) {
				return n, invalidAnimation()
			}
			pos++
			if !skipBlocks() {
				return n, invalidAnimation()
			}
		case 0x2c:
			if len(b)-pos < 9 {
				return n, invalidAnimation()
			}
			x := int(binary.LittleEndian.Uint16(b[pos : pos+2]))
			y := int(binary.LittleEndian.Uint16(b[pos+2 : pos+4]))
			w := int(binary.LittleEndian.Uint16(b[pos+4 : pos+6]))
			h := int(binary.LittleEndian.Uint16(b[pos+6 : pos+8]))
			if w == 0 || h == 0 || x+w > int(binary.LittleEndian.Uint16(b[6:8])) || y+h > int(binary.LittleEndian.Uint16(b[8:10])) {
				return n, invalidAnimation()
			}
			packed := b[pos+8]
			pos += 9
			if packed&128 != 0 {
				pos += 3 * (1 << ((packed & 7) + 1))
			}
			if pos >= len(b) {
				return n, invalidAnimation()
			}
			pos++
			if !skipBlocks() {
				return n, invalidAnimation()
			}
			n++
			if limit < 0 {
				return n, nil
			}
			if int64(n) > limit {
				return n, frameLimit()
			}
		default:
			return n, invalidAnimation()
		}
	}
	return n, invalidAnimation()
}

func uint24(b []byte) int { return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 }
