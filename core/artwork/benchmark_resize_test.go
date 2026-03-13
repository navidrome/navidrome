package artwork

import (
	"fmt"
	"testing"

	"github.com/disintegration/imaging"
)

func BenchmarkImageResize(b *testing.B) {
	sourceSizes := []int{300, 1000, 3000}
	targetSizes := []int{300, 600}

	for _, srcSize := range sourceSizes {
		img := generateGradientImage(srcSize, srcSize)
		for _, tgtSize := range targetSizes {
			if tgtSize >= srcSize {
				continue // Skip upscaling (not a valid resize scenario)
			}
			b.Run(fmt.Sprintf("%dx%d_to_%d", srcSize, srcSize, tgtSize), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					imaging.Fit(img, tgtSize, tgtSize, imaging.Lanczos)
				}
			})
		}
	}
}
