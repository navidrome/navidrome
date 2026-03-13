package artwork

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"testing"
)

func BenchmarkImageDecode(b *testing.B) {
	sizes := []int{300, 1000, 3000}
	formats := []struct {
		name string
		gen  func(tb testing.TB, w, h int) []byte
	}{
		{"jpeg", func(tb testing.TB, w, h int) []byte { return generateJPEG(tb, w, h, 75) }},
		{"png", func(tb testing.TB, w, h int) []byte { return generatePNG(tb, w, h) }},
	}

	for _, format := range formats {
		for _, size := range sizes {
			data := format.gen(b, size, size)
			b.Run(fmt.Sprintf("%s/%dx%d", format.name, size, size), func(b *testing.B) {
				b.SetBytes(int64(len(data)))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _, err := image.Decode(bytes.NewReader(data))
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
