package processing

import "image-processing-worker/application/ports"

func CenterCrop(width, height int, reference ports.Reference) ports.Rectangle {
	w, h := width, height
	if int64(width)*int64(reference.Height) > int64(height)*int64(reference.Width) {
		w = int(int64(height) * int64(reference.Width) / int64(reference.Height))
	} else {
		h = int(int64(width) * int64(reference.Height) / int64(reference.Width))
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return ports.Rectangle{X: (width - w) / 2, Y: (height - h) / 2, Width: w, Height: h}
}
func outputSize(c ports.Rectangle, r ports.Reference) (int, int) {
	w, h := c.Width, c.Height
	if w > r.Width {
		h = max(1, int(int64(h)*int64(r.Width)/int64(w)))
		w = r.Width
	}
	if h > r.Height {
		w = max(1, int(int64(w)*int64(r.Height)/int64(h)))
		h = r.Height
	}
	return w, h
}
