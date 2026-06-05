package artwork

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"image/png"
	"testing"
)

func BenchmarkImageEncode(b *testing.B) {
	img := generateGradientImage(300, 300)

	jpegQualities := []int{60, 75, 90}
	for _, q := range jpegQualities {
		b.Run(fmt.Sprintf("jpeg/q%d/300x300", q), func(b *testing.B) {
			var buf bytes.Buffer
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf.Reset()
				if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(buf.Len()), "bytes")
		})
	}

	b.Run("png/300x300", func(b *testing.B) {
		var buf bytes.Buffer
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			if err := png.Encode(&buf, img); err != nil {
				b.Fatal(err)
			}
		}
		b.ReportMetric(float64(buf.Len()), "bytes")
	})
}
