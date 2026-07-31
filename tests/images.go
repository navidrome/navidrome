package tests

import (
	"image"
	"image/color"
)

// GradientImage builds a deterministic square gradient, so benchmark runs are comparable across
// revisions. NRGBA is the type the artwork pipeline's makeThumbnail hands its hash encoders.
func GradientImage(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(255 * x / size),
				G: uint8(255 * y / size),
				B: uint8((x + y) * 255 / (2 * size)),
				A: 255,
			})
		}
	}
	return img
}
