package artwork

import (
	"fmt"
	"image"
	"testing"

	"github.com/navidrome/navidrome/core/artwork/blurhash"
	"github.com/navidrome/navidrome/core/artwork/thumbhash"
	"github.com/navidrome/navidrome/tests"
)

// hashEncoders are the two placeholder hashes decodeArtwork computes from one shared thumbnail.
var hashEncoders = []struct {
	name   string
	encode func(image.Image) error
}{
	{"blurhash", func(img image.Image) error { _, err := blurhash.Encode(img); return err }},
	{"thumbhash", func(img image.Image) error { _, err := thumbhash.Encode(img); return err }},
}

func benchEncoder(b *testing.B, encode func(image.Image) error, img image.Image) {
	b.Helper()
	b.ReportAllocs()
	for b.Loop() {
		if err := encode(img); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHashEncodersAtInputSize is the bar: both encoders are handed the identical image
// makeThumbnail produces, so neither is measured with a conversion the other avoids.
func BenchmarkHashEncodersAtInputSize(b *testing.B) {
	img := tests.GradientImage(thumbnailSize)
	for _, e := range hashEncoders {
		b.Run(e.name, func(b *testing.B) { benchEncoder(b, e.encode, img) })
	}
}

// BenchmarkHashEncoders sweeps past the pipeline's input size, where each package's own defensive
// downscale starts to dominate.
func BenchmarkHashEncoders(b *testing.B) {
	for _, size := range []int{100, 300, 600, 900, 1200, 1500} {
		img := tests.GradientImage(size)
		for _, e := range hashEncoders {
			b.Run(fmt.Sprintf("%s/%dx%d", e.name, size, size), func(b *testing.B) {
				benchEncoder(b, e.encode, img)
			})
		}
	}
}
