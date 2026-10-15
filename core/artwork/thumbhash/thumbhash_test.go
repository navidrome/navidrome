package thumbhash_test

import (
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/navidrome/navidrome/core/artwork/thumbhash"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testdataDir is resolved via runtime.Caller because tests.Init (thumbhash_suite_test.go) chdirs
// the process to the repo root, which would break a plain relative "testdata" path.
var testdataDir = func() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}()

// fixtureImage decodes a testdata PNG. NRGBA, not RGBA: ThumbHash needs non-premultiplied pixels,
// and RGBA would silently premultiply every fixture that has alpha.
func fixtureImage(name string) *image.NRGBA {
	GinkgoHelper()
	f, err := os.Open(filepath.Join(testdataDir, name))
	Expect(err).ToNot(HaveOccurred())
	defer f.Close()
	src, _, err := image.Decode(f)
	Expect(err).ToNot(HaveOccurred())
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

// headerOnlyFixtures have mathematically-zero AC content, so every AC nibble is float rounding
// noise sitting on a quantization tie; only the header bytes carry signal.
var headerOnlyFixtures = []string{"solid.png", "tiny.png"}

func loadJSON[T any](name string) T {
	GinkgoHelper()
	data, err := os.ReadFile(filepath.Join(testdataDir, name))
	Expect(err).ToNot(HaveOccurred())
	var out T
	Expect(json.Unmarshal(data, &out)).To(Succeed())
	return out
}

func loadGoldens() map[string]string {
	GinkgoHelper()
	golden := loadJSON[map[string]string]("golden.json")
	Expect(golden).ToNot(BeEmpty())
	return golden
}

// mix is a stateless 32-bit finaliser matching gen_generated.mjs, so both languages build the
// same synthetic pixels and only the reference's hashes need committing.
func mix(n uint32) uint32 {
	n = (n ^ (n >> 16)) * 2246822507
	n = (n ^ (n >> 13)) * 3266489909
	return n ^ (n >> 16)
}

func generatedImage(i int) *image.NRGBA {
	w := 1 + int(mix(uint32(i)*3+1)%100)
	h := 1 + int(mix(uint32(i)*3+2)%100)
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for k := range img.Pix {
		img.Pix[k] = byte(mix(uint32(i)*1000003 + uint32(k)))
	}
	if i%2 == 0 {
		for k := 3; k < len(img.Pix); k += 4 {
			img.Pix[k] = 255
		}
	}
	return img
}

var _ = Describe("Encode", func() {
	It("matches every golden vector", func() {
		for name, want := range loadGoldens() {
			if slices.Contains(headerOnlyFixtures, name) {
				continue // see the dedicated header-only spec below
			}
			got, err := thumbhash.Encode(fixtureImage(name))
			Expect(err).ToNot(HaveOccurred(), "fixture %s", name)
			Expect(base64.StdEncoding.EncodeToString(got)).To(Equal(want), "fixture %s", name)
		}
	})

	It("reproduces the well-conditioned header of the ill-conditioned fixtures", func() {
		for _, name := range headerOnlyFixtures {
			want, err := base64.StdEncoding.DecodeString(loadGoldens()[name])
			Expect(err).ToNot(HaveOccurred(), "fixture %s", name)
			got, err := thumbhash.Encode(fixtureImage(name))
			Expect(err).ToNot(HaveOccurred(), "fixture %s", name)
			Expect(got[:5]).To(Equal(want[:5]), "fixture %s header bytes", name)
		}
	})

	// The PNG fixtures cannot reach every layout; these sweep random sizes, aspects and both the
	// 7x7 no-alpha and 5x5-plus-alpha coefficient regions against the same reference.
	It("matches the reference on 300 generated images", func() {
		vectors := loadJSON[[]struct {
			W, H int
			Hash string
		}]("generated.json")
		Expect(vectors).ToNot(BeEmpty())
		for i, want := range vectors {
			img := generatedImage(i)
			Expect(img.Bounds().Dx()).To(Equal(want.W), "vector %d width", i)
			Expect(img.Bounds().Dy()).To(Equal(want.H), "vector %d height", i)
			got, err := thumbhash.Encode(img)
			Expect(err).ToNot(HaveOccurred(), "vector %d", i)
			Expect(base64.StdEncoding.EncodeToString(got)).To(Equal(want.Hash), "vector %d (%dx%d)", i, want.W, want.H)
		}
	})

	It("quantizes a uniform image's scales to zero", func() {
		got, err := thumbhash.Encode(fixtureImage("solid.png"))
		Expect(err).ToNot(HaveOccurred())
		header24 := int(got[0]) | int(got[1])<<8 | int(got[2])<<16
		header16 := int(got[3]) | int(got[4])<<8
		Expect((header24>>18)&31).To(Equal(0), "lScale")
		Expect((header16>>3)&63).To(Equal(0), "pScale")
		Expect((header16>>9)&63).To(Equal(0), "qScale")
	})

	It("produces 24 bytes for a square opaque image", func() {
		got, err := thumbhash.Encode(fixtureImage("square.png"))
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(HaveLen(24))
	})

	It("produces 25 bytes when the image has alpha", func() {
		got, err := thumbhash.Encode(fixtureImage("alpha.png"))
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(HaveLen(25))
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

	It("encodes a sub-image like an origin-anchored copy of the same region", func() {
		parent := image.NewNRGBA(image.Rect(0, 0, 60, 50))
		for i := range parent.Pix {
			parent.Pix[i] = byte(i * 7 % 251)
		}
		region := image.Rect(10, 7, 40, 30)
		cropped := image.NewNRGBA(image.Rect(0, 0, region.Dx(), region.Dy()))
		draw.Draw(cropped, cropped.Bounds(), parent, region.Min, draw.Src)

		got, err := thumbhash.Encode(parent.SubImage(region))
		Expect(err).ToNot(HaveOccurred())
		want, err := thumbhash.Encode(cropped)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal(want))
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
