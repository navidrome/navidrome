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
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func gradientImage(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(255 * x / w), G: uint8(255 * y / h), B: 128, A: 255})
		}
	}
	return img
}

var _ = Describe("Encode input types", func() {
	// The pipeline hands Encode an *image.NRGBA; reading it must stay equivalent to the
	// premultiplied *image.RGBA it used to receive, or every hash silently shifts.
	buildPair := func(alpha uint8) (*image.NRGBA, *image.RGBA) {
		const size = 40
		nrgba := image.NewNRGBA(image.Rect(0, 0, size, size))
		rgba := image.NewRGBA(image.Rect(0, 0, size, size))
		for y := range size {
			for x := range size {
				c := color.NRGBA{
					R: uint8(255 * x / size), G: uint8(255 * y / size),
					B: uint8((x + y) * 255 / (2 * size)), A: alpha,
				}
				nrgba.SetNRGBA(x, y, c)
				rgba.Set(x, y, c) // image.RGBA.Set premultiplies
			}
		}
		return nrgba, rgba
	}

	It("gives an opaque NRGBA the same hash as the equivalent RGBA", func() {
		nrgba, rgba := buildPair(255)
		fromNRGBA, err := blurhash.Encode(nrgba)
		Expect(err).ToNot(HaveOccurred())
		fromRGBA, err := blurhash.Encode(rgba)
		Expect(err).ToNot(HaveOccurred())
		Expect(fromNRGBA).To(Equal(fromRGBA))
	})

	It("premultiplies a partly transparent NRGBA, matching the RGBA it replaces", func() {
		nrgba, rgba := buildPair(128)
		fromNRGBA, err := blurhash.Encode(nrgba)
		Expect(err).ToNot(HaveOccurred())
		fromRGBA, err := blurhash.Encode(rgba)
		Expect(err).ToNot(HaveOccurred())
		Expect(fromNRGBA).To(Equal(fromRGBA))
	})

	It("treats a fully transparent NRGBA as black, as premultiplication does", func() {
		nrgba, rgba := buildPair(0)
		fromNRGBA, err := blurhash.Encode(nrgba)
		Expect(err).ToNot(HaveOccurred())
		fromRGBA, err := blurhash.Encode(rgba)
		Expect(err).ToNot(HaveOccurred())
		Expect(fromNRGBA).To(Equal(fromRGBA))
	})
})

var _ = Describe("Encode", func() {
	// The size flag encodes (xComp-1) + (yComp-1)*9.
	DescribeTable("derives component counts from aspect ratio (Jellyfin formula)",
		func(w, h, expectedX, expectedY int) {
			hash, err := blurhash.Encode(gradientImage(w, h))
			Expect(err).ToNot(HaveOccurred())
			Expect(decode83(hash[:1])).To(Equal((expectedX - 1) + (expectedY-1)*9))
		},
		Entry("square album art", 60, 60, 5, 5),
		Entry("smallest square", 1, 1, 5, 5),
		Entry("landscape 16:9", 192, 108, 6, 4),
		Entry("portrait 9:16", 108, 192, 4, 6),
		Entry("extreme landscape capped at 9", 1000, 10, 9, 1),
		Entry("extreme portrait capped at 9", 10, 1000, 1, 9),
	)

	It("rejects an empty image", func() {
		_, err := blurhash.Encode(image.NewNRGBA(image.Rect(0, 0, 0, 0)))
		Expect(err).To(HaveOccurred())
	})

	It("produces the spec-mandated length", func() {
		// 1 (size flag) + 1 (max AC) + 4 (DC) + 2 per AC component; a square derives 5x5
		h, err := blurhash.Encode(solidImage(8, 8, color.NRGBA{R: 10, G: 20, B: 30, A: 255}))
		Expect(err).ToNot(HaveOccurred())
		Expect(h).To(HaveLen(4 + 2 + 2*(5*5-1)))
	})

	It("stores the average color in the DC component", func() {
		h, err := blurhash.Encode(solidImage(16, 16, color.NRGBA{R: 200, G: 100, B: 50, A: 255}))
		Expect(err).ToNot(HaveOccurred())
		dc := decode83(h[2:6])
		Expect(dc >> 16).To(BeNumerically("~", 200, 1))
		Expect((dc >> 8) & 0xFF).To(BeNumerically("~", 100, 1))
		Expect(dc & 0xFF).To(BeNumerically("~", 50, 1))
	})

	It("is deterministic", func() {
		img := gradientImage(64, 64)
		h1, err1 := blurhash.Encode(img)
		h2, err2 := blurhash.Encode(img)
		Expect(err1).ToNot(HaveOccurred())
		Expect(err2).ToNot(HaveOccurred())
		Expect(h1).To(Equal(h2))
	})

	It("produces different hashes for different images", func() {
		h1, _ := blurhash.Encode(solidImage(16, 16, color.NRGBA{R: 255, A: 255}))
		h2, _ := blurhash.Encode(gradientImage(16, 16))
		Expect(h1).ToNot(Equal(h2))
	})

	It("downscales large images internally without changing the result materially", func() {
		big, err := blurhash.Encode(solidImage(1000, 1000, color.NRGBA{R: 60, G: 120, B: 180, A: 255}))
		Expect(err).ToNot(HaveOccurred())
		small, err := blurhash.Encode(solidImage(16, 16, color.NRGBA{R: 60, G: 120, B: 180, A: 255}))
		Expect(err).ToNot(HaveOccurred())
		Expect(big[2:6]).To(Equal(small[2:6]))
	})
})
