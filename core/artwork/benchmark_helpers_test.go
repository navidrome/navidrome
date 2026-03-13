package artwork

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// generateJPEG creates a JPEG image of the given dimensions with a gradient pattern.
// The gradient ensures the image has realistic entropy (not trivially compressible).
func generateJPEG(t testing.TB, width, height, quality int) []byte {
	t.Helper()
	img := generateGradientImage(width, height)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// generatePNG creates a PNG image of the given dimensions with a gradient pattern.
func generatePNG(t testing.TB, width, height int) []byte {
	t.Helper()
	img := generateGradientImage(width, height)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// generateGradientImage creates an RGBA image with a diagonal gradient pattern.
func generateGradientImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(((x + y) * 255) / (width + height))
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}
