package blurhash_test

import (
	"image"
	"image/color"
	"strings"

	"github.com/navidrome/navidrome/core/artwork/blurhash"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz#$%*+,-.:;=?@[]^_{|}~"

func decode83(s string) int {
	v := 0
	for _, c := range s {
		v = v*83 + strings.IndexRune(alphabet, c)
	}
	return v
}

func solidImage(w, h int, c color.NRGBA) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func gradientImage(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(255 * x / w), G: uint8(255 * y / h), B: 128, A: 255})
		}
	}
	return img
}

var _ = Describe("Components", func() {
	DescribeTable("derives component counts from aspect ratio (Jellyfin formula)",
		func(w, h, expectedX, expectedY int) {
			x, y := blurhash.Components(w, h)
			Expect(x).To(Equal(expectedX))
			Expect(y).To(Equal(expectedY))
		},
		Entry("square album art", 600, 600, 5, 5),
		Entry("small square", 1, 1, 5, 5),
		Entry("landscape 16:9", 1920, 1080, 6, 4),
		Entry("portrait 9:16", 1080, 1920, 4, 6),
		Entry("extreme landscape capped at 9", 10000, 100, 9, 1),
		Entry("zero width", 0, 600, 0, 0),
		Entry("zero height", 600, 0, 0, 0),
	)
})

var _ = Describe("Encode", func() {
	It("rejects out-of-range components", func() {
		_, err := blurhash.Encode(solidImage(8, 8, color.NRGBA{A: 255}), 0, 5)
		Expect(err).To(HaveOccurred())
		_, err = blurhash.Encode(solidImage(8, 8, color.NRGBA{A: 255}), 5, 10)
		Expect(err).To(HaveOccurred())
	})

	It("produces the spec-mandated length", func() {
		// 1 (size flag) + 1 (max AC) + 4 (DC) + 2 per AC component
		h, err := blurhash.Encode(solidImage(8, 8, color.NRGBA{R: 10, G: 20, B: 30, A: 255}), 4, 3)
		Expect(err).ToNot(HaveOccurred())
		Expect(h).To(HaveLen(4 + 2 + 2*(4*3-1)))
	})

	It("encodes the size flag as the first character", func() {
		h, err := blurhash.Encode(solidImage(8, 8, color.NRGBA{A: 255}), 4, 3)
		Expect(err).ToNot(HaveOccurred())
		Expect(decode83(h[:1])).To(Equal((4 - 1) + (3-1)*9))
	})

	It("stores the average color in the DC component", func() {
		h, err := blurhash.Encode(solidImage(16, 16, color.NRGBA{R: 200, G: 100, B: 50, A: 255}), 4, 3)
		Expect(err).ToNot(HaveOccurred())
		dc := decode83(h[2:6])
		Expect(dc >> 16).To(BeNumerically("~", 200, 1))
		Expect((dc >> 8) & 0xFF).To(BeNumerically("~", 100, 1))
		Expect(dc & 0xFF).To(BeNumerically("~", 50, 1))
	})

	It("is deterministic", func() {
		img := gradientImage(64, 64)
		h1, err1 := blurhash.Encode(img, 5, 5)
		h2, err2 := blurhash.Encode(img, 5, 5)
		Expect(err1).ToNot(HaveOccurred())
		Expect(err2).ToNot(HaveOccurred())
		Expect(h1).To(Equal(h2))
	})

	It("produces different hashes for different images", func() {
		h1, _ := blurhash.Encode(solidImage(16, 16, color.NRGBA{R: 255, A: 255}), 4, 4)
		h2, _ := blurhash.Encode(gradientImage(16, 16), 4, 4)
		Expect(h1).ToNot(Equal(h2))
	})

	It("downscales large images internally without changing the result materially", func() {
		// A 1000px solid image must encode fine and carry the same DC as its small version.
		big, err := blurhash.Encode(solidImage(1000, 1000, color.NRGBA{R: 60, G: 120, B: 180, A: 255}), 5, 5)
		Expect(err).ToNot(HaveOccurred())
		small, err := blurhash.Encode(solidImage(16, 16, color.NRGBA{R: 60, G: 120, B: 180, A: 255}), 5, 5)
		Expect(err).ToNot(HaveOccurred())
		Expect(big[2:6]).To(Equal(small[2:6]))
	})
})
