package image_processor

type InfoCrop struct {
	PictureWidth  int
	PictureHeight int
	MaxWidth      int
	MaxHeight     int
}

func Crop(info InfoCrop) (width, height int) {

	if info.PictureWidth*info.MaxHeight > info.PictureHeight*info.MaxWidth {
		height = info.PictureHeight
		width = (info.MaxWidth * height) / info.MaxHeight

		return width, height
	} else if info.PictureWidth*info.MaxHeight < info.PictureHeight*info.MaxWidth {
		width = info.PictureWidth
		height = (width * info.MaxHeight) / info.MaxWidth

		return width, height
	}

	return info.PictureWidth, info.PictureHeight
}
