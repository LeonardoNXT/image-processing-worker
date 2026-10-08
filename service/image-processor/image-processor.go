package image_processor

import (
	"image"
	imageDecoder "image-parsing-worker/service/image-decoder"
	"image-parsing-worker/structs"
)

type Position struct {
	CordX string
	CordY string
}

type Animation struct {
	Images    []*image.RGBA
	Positions []image.Rectangle
}

type Image struct {
	Image    *image.RGBA
	Position image.Rectangle
}

type OutputProcessor struct {
	ImageProcessed     Image
	AnimationProcessed Animation
}

func getSize(dx int, dy int, shape structs.OutputShape) (w int, h int) {

}

func getCropPosition(dx int, dy int, shape structs.OutputShape) image.Rectangle {

}

func Exec(decodedImage imageDecoder.DecodedImage, shape structs.OutputShape) (output OutputProcessor, err error) {

	dx := decodedImage.Image.Bounds().Dx()
	dy := decodedImage.Image.Bounds().Dy()

}
