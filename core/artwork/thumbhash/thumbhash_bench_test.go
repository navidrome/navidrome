package thumbhash_test

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/navidrome/navidrome/core/artwork/thumbhash"
)

// benchImage builds the same deterministic gradient the blurhash bench uses, as *image.RGBA —
// the concrete type makeThumbnail hands both encoders in production.
func benchImage(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(255 * x / size),
				G: uint8(255 * y / size),
				B: uint8((x + y) * 255 / (2 * size)),
				A: 255,
			})
		}
	}
	return img
}

// BenchmarkEncodeAtInputSize is the bar: 100x100 is exactly what the artwork pipeline feeds.
func BenchmarkEncodeAtInputSize(b *testing.B) {
	img := benchImage(100)
	b.ReportAllocs()
	for range b.N {
		if _, err := thumbhash.Encode(img); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	for _, size := range []int{100, 300, 600, 900, 1200, 1500} {
		img := benchImage(size)
		b.Run(fmt.Sprintf("%dx%d", size, size), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if _, err := thumbhash.Encode(img); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkReferenceEncode is the naive baseline: four w*h scratch arrays and 40 pixel re-reads.
func BenchmarkReferenceEncode(b *testing.B) {
	img := benchImage(100)
	w, h := 100, 100
	pix := make([]byte, 0, w*h*4)
	for y := range h {
		pix = append(pix, img.Pix[y*img.Stride:y*img.Stride+w*4]...)
	}
	b.ReportAllocs()
	for range b.N {
		_ = referenceEncode(w, h, pix)
	}
}
