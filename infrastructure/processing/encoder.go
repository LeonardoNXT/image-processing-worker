package processing

import (
	"encoding/binary"
	"image"
	"io"
)

func encodedMetadata(r io.ReadSeeker, format string) (int, int, int, error) {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return 0, 0, 0, err
	}
	frames := 1
	if format == "webp" {
		if _, err = r.Seek(12, io.SeekStart); err != nil {
			return 0, 0, 0, err
		}
		count := 0
		for {
			var header [8]byte
			_, err = io.ReadFull(r, header[:])
			if err == io.EOF {
				break
			}
			if err != nil {
				return 0, 0, 0, err
			}
			if string(header[:4]) == "ANMF" {
				count++
			}
			size := int64(binary.LittleEndian.Uint32(header[4:]))
			size += size % 2
			if _, err = r.Seek(size, io.SeekCurrent); err != nil {
				return 0, 0, 0, err
			}
		}
		frames = max(1, count)
	}
	_, err = r.Seek(0, io.SeekStart)
	return cfg.Width, cfg.Height, frames, err
}
