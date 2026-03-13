package artwork

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
)

func BenchmarkResizeFullPipeline(b *testing.B) {
	cleanup := configtest.SetupConfig()
	b.Cleanup(cleanup)
	conf.Server.CoverArtQuality = 75

	sourceSizes := []int{1000, 3000}
	targetSize := 300

	for _, srcSize := range sourceSizes {
		jpegData := generateJPEG(b, srcSize, srcSize, 90)

		b.Run(fmt.Sprintf("jpeg/%dx%d_to_%d", srcSize, srcSize, targetSize), func(b *testing.B) {
			b.SetBytes(int64(len(jpegData)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, _, err := resizeImage(bytes.NewReader(jpegData), targetSize, false)
				if err != nil {
					b.Fatal(err)
				}
				if result == nil {
					b.Fatal("expected non-nil resized image")
				}
			}
		})

		b.Run(fmt.Sprintf("jpeg/%dx%d_to_%d_square", srcSize, srcSize, targetSize), func(b *testing.B) {
			b.SetBytes(int64(len(jpegData)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, _, err := resizeImage(bytes.NewReader(jpegData), targetSize, true)
				if err != nil {
					b.Fatal(err)
				}
				if result == nil {
					b.Fatal("expected non-nil resized image")
				}
			}
		})
	}
}
