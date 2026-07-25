package thumbhash_test

import (
	"encoding/base64"
	"image"
	"image/color"
	"math/rand/v2"

	"github.com/navidrome/navidrome/core/artwork/thumbhash"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fixtureImage rebuilds a testdata PNG as an image.Image for the Encode API. NRGBA, not RGBA:
// the pixels are non-premultiplied and must stay that way.
func fixtureImage(name string) image.Image {
	GinkgoHelper()
	w, h, pix := loadFixture(name)
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		copy(img.Pix[y*img.Stride:], pix[y*w*4:(y+1)*w*4])
	}
	return img
}

var _ = Describe("Encode", func() {
	It("matches every golden vector", func() {
		for name, want := range loadGoldens() {
			if name == "solid.png" {
				continue // see the dedicated header-only spec below
			}
			got, err := thumbhash.Encode(fixtureImage(name))
			Expect(err).ToNot(HaveOccurred(), "fixture %s", name)
			Expect(base64.StdEncoding.EncodeToString(got)).To(Equal(want), "fixture %s", name)
		}
	})

	// A uniform image has mathematically-zero AC terms, so its AC nibbles are rounding noise
	// normalized by a scale that is itself noise; only the header is well-defined.
	It("reproduces the well-conditioned header of a uniform image", func() {
		want, err := base64.StdEncoding.DecodeString(loadGoldens()["solid.png"])
		Expect(err).ToNot(HaveOccurred())
		got, err := thumbhash.Encode(fixtureImage("solid.png"))
		Expect(err).ToNot(HaveOccurred())
		Expect(got[:5]).To(Equal(want[:5]), "header bytes")
	})

	It("agrees with the reference port on randomized images", func() {
		rng := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a fixed seed is the point: the run must be reproducible
		for range 500 {
			w := 1 + rng.IntN(100)
			h := 1 + rng.IntN(100)
			img := image.NewNRGBA(image.Rect(0, 0, w, h))
			for i := range img.Pix {
				img.Pix[i] = byte(rng.IntN(256))
			}
			// Alpha is randomized too, so the 5x5-plus-alpha layout is exercised as often as 7x7.
			got, err := thumbhash.Encode(img)
			Expect(err).ToNot(HaveOccurred())

			pix := make([]byte, 0, w*h*4)
			for y := range h {
				pix = append(pix, img.Pix[y*img.Stride:y*img.Stride+w*4]...)
			}
			Expect(got).To(Equal(referenceEncode(w, h, pix)), "%dx%d", w, h)
		}
	})

	It("downscales an oversized image rather than failing", func() {
		img := image.NewNRGBA(image.Rect(0, 0, 500, 300))
		for i := range img.Pix {
			img.Pix[i] = byte(i)
		}
		got, err := thumbhash.Encode(img)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(got)).To(BeNumerically(">=", 5))
	})

	It("encodes a 1x1 image", func() {
		img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		img.Set(0, 0, color.NRGBA{R: 60, G: 120, B: 180, A: 255})
		got, err := thumbhash.Encode(img)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).ToNot(BeEmpty())
	})

	It("rejects an empty image", func() {
		_, err := thumbhash.Encode(image.NewRGBA(image.Rect(0, 0, 0, 0)))
		Expect(err).To(HaveOccurred())
	})

	It("is deterministic", func() {
		img := fixtureImage("square.png")
		first, err := thumbhash.Encode(img)
		Expect(err).ToNot(HaveOccurred())
		second, err := thumbhash.Encode(img)
		Expect(err).ToNot(HaveOccurred())
		Expect(first).To(Equal(second))
	})
})
