package thumbhash_test

import (
	"testing"

	"github.com/navidrome/navidrome/core/artwork/thumbhash"
	"github.com/navidrome/navidrome/tests"
)

// BenchmarkEncodeVsReference pins the two-pass separable encoder against the naive reference port
// it replaced. referenceEncode is test-only, so this cannot live beside the cross-encoder
// benchmark in core/artwork.
func BenchmarkEncodeVsReference(b *testing.B) {
	const size = 100
	img := tests.GradientImage(size)
	pix := make([]byte, 0, size*size*4)
	for y := range size {
		pix = append(pix, img.Pix[y*img.Stride:y*img.Stride+size*4]...)
	}

	b.Run("shipped", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := thumbhash.Encode(img); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("reference", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = referenceEncode(size, size, pix)
		}
	})
}
