package imageDecoder

import (
	"fmt"
	"image"
	"image/gif"
	"slices"

	"github.com/minio/minio-go/v7"
)

type DecodedImage struct {
	Image     image.Image
	Animation *gif.GIF
}

func Exec(object *minio.Object, permittedFormats []string) (format string, src DecodedImage, err error) {
	defer object.Close()

	img, format, DecoderErr := image.Decode(object)

	if DecoderErr != nil {
		return "", DecodedImage{}, DecoderErr
	}

	if !slices.Contains(permittedFormats, format) {
		return "", DecodedImage{}, fmt.Errorf("format %s not supported", format)
	}

	if format == "gif" {

		_, seekErr := object.Seek(0, 0)

		if seekErr != nil {
			return "", DecodedImage{}, seekErr
		}

		gifSrc, err := gif.DecodeAll(object)

		if err != nil {
			return "", DecodedImage{}, err
		}

		return format, DecodedImage{
			Image:     nil,
			Animation: gifSrc,
		}, nil

	}

	return format, DecodedImage{
		Image:     img,
		Animation: nil,
	}, nil
}
