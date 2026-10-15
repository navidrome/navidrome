package dominant_test

import (
	"image"
	"image/color"

	"github.com/navidrome/navidrome/core/artwork/dominant"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fill paints rect with c onto img.
func fill(img *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

func newImg(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	fill(img, img.Bounds(), c)
	return img
}

var _ = Describe("Color", func() {
	It("returns a solid image's own colour", func() {
		Expect(dominant.Color(newImg(20, 20, color.NRGBA{0x33, 0x66, 0x99, 255}))).To(Equal("#336699"))
	})

	It("returns empty for an image with no pixels", func() {
		Expect(dominant.Color(image.NewNRGBA(image.Rect(0, 0, 0, 0)))).To(Equal(""))
	})

	// Presence, not salience: this is a placeholder, so the large field wins even though the small
	// patch is the more interesting colour.
	It("picks the largest area, not the most vivid one", func() {
		img := newImg(20, 20, color.NRGBA{0xfa, 0xfa, 0xfa, 255})
		fill(img, image.Rect(0, 0, 4, 4), color.NRGBA{0xff, 0x00, 0x00, 255})
		Expect(dominant.Color(img)).To(Equal("#fafafa"))
	})

	It("reports a near-black cover as near-black", func() {
		img := newImg(20, 20, color.NRGBA{0x05, 0x05, 0x05, 255})
		fill(img, image.Rect(0, 0, 5, 5), color.NRGBA{0x00, 0xff, 0x00, 255})
		Expect(dominant.Color(img)).To(Equal("#050505"))
	})

	// A gradient splits across many quantisation bins. Without merging, each slice is smaller than
	// the flat block and the block would win despite covering far less of the image.
	It("merges a gradient's bins so it beats a smaller flat block", func() {
		img := image.NewNRGBA(image.Rect(0, 0, 40, 40))
		for y := range 40 {
			for x := range 40 {
				// 30 columns of blue gradient == 75% of the image
				if x < 30 {
					img.SetNRGBA(x, y, color.NRGBA{0x10, 0x20, uint8(0xa0 + x), 255})
				} else {
					img.SetNRGBA(x, y, color.NRGBA{0xff, 0xcc, 0x00, 255})
				}
			}
		}
		got := dominant.Color(img)
		Expect(got).To(HavePrefix("#1020"), "expected the blue gradient, got "+got)
	})

	It("is deterministic", func() {
		img := image.NewNRGBA(image.Rect(0, 0, 30, 30))
		for y := range 30 {
			for x := range 30 {
				img.SetNRGBA(x, y, color.NRGBA{uint8(x * 7), uint8(y * 5), uint8(x + y), 255})
			}
		}
		first := dominant.Color(img)
		for range 5 {
			Expect(dominant.Color(img)).To(Equal(first))
		}
	})

	It("handles images that are not NRGBA", func() {
		src := newImg(10, 10, color.NRGBA{0x20, 0x40, 0x60, 255})
		rgba := image.NewRGBA(src.Bounds())
		for y := range 10 {
			for x := range 10 {
				rgba.Set(x, y, src.At(x, y))
			}
		}
		Expect(dominant.Color(rgba)).To(Equal("#204060"))
	})
})
