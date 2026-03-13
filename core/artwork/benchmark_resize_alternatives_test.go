package artwork

import (
	"fmt"
	"image"
	"image/draw"
	"testing"

	"github.com/disintegration/imaging"
	xdraw "golang.org/x/image/draw"
)

func BenchmarkResizeAlternatives(b *testing.B) {
	sourceSizes := []int{1000, 3000}
	targetSize := 300

	resizers := []struct {
		name   string
		resize func(src image.Image, targetSize int) image.Image
	}{
		{"imaging_lanczos", func(src image.Image, ts int) image.Image {
			return imaging.Fit(src, ts, ts, imaging.Lanczos)
		}},
		{"imaging_catmullrom", func(src image.Image, ts int) image.Image {
			return imaging.Fit(src, ts, ts, imaging.CatmullRom)
		}},
		{"imaging_linear", func(src image.Image, ts int) image.Image {
			return imaging.Fit(src, ts, ts, imaging.Linear)
		}},
		{"xdraw_catmullrom", func(src image.Image, ts int) image.Image {
			dst := image.NewRGBA(image.Rect(0, 0, ts, ts))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
			return dst
		}},
		{"xdraw_approxbilinear", func(src image.Image, ts int) image.Image {
			dst := image.NewRGBA(image.Rect(0, 0, ts, ts))
			xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
			return dst
		}},
	}

	for _, srcSize := range sourceSizes {
		src := generateGradientImage(srcSize, srcSize)
		for _, r := range resizers {
			b.Run(fmt.Sprintf("%s/%dx%d_to_%d", r.name, srcSize, srcSize, targetSize), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					r.resize(src, targetSize)
				}
			})
		}
	}
}
