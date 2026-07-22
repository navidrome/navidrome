package blurhash_test

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/navidrome/navidrome/core/artwork/blurhash"
)

// benchImage builds a deterministic gradient so runs are comparable across revisions.
func benchImage(size int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
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

func BenchmarkEncode(b *testing.B) {
	for _, size := range []int{100, 300, 600, 900, 1200, 1500} {
		img := benchImage(size)
		x, y := blurhash.Components(size, size)
		b.Run(fmt.Sprintf("%dx%d", size, size), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if _, err := blurhash.Encode(img, x, y); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
