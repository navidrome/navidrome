package artwork

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func benchJPEG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(255 * x / size), G: uint8(255 * y / size),
				B: uint8((x + y) * 255 / (2 * size)), A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// BenchmarkDecodeArtwork measures the whole per-image cost the artwork worker pays.
func BenchmarkDecodeArtwork(b *testing.B) {
	data := benchJPEG(1000)
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := decodeArtwork(ctx, "bench", data); err != nil {
			b.Fatal(err)
		}
	}
}
