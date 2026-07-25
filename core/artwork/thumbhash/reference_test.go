package thumbhash_test

import (
	"encoding/base64"
	"encoding/json"
	"image"
	"image/draw"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testdataDir is resolved via runtime.Caller because tests.Init (thumbhash_suite_test.go) chdirs
// the process to the repo root, which would break a plain relative "testdata" path.
var testdataDir = func() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}()

// loadFixture returns a testdata PNG as tightly-packed NON-premultiplied RGBA, which is what the
// reference JS sees when it reads the raw PNG bytes. image.NRGBA is the non-premultiplied type;
// image.RGBA would silently premultiply and break every fixture that has alpha.
func loadFixture(name string) (int, int, []byte) {
	GinkgoHelper()
	f, err := os.Open(filepath.Join(testdataDir, name))
	Expect(err).ToNot(HaveOccurred())
	defer f.Close()
	src, _, err := image.Decode(f)
	Expect(err).ToNot(HaveOccurred())
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	pix := make([]byte, 0, b.Dx()*b.Dy()*4)
	for y := range b.Dy() {
		pix = append(pix, dst.Pix[y*dst.Stride:y*dst.Stride+b.Dx()*4]...)
	}
	return b.Dx(), b.Dy(), pix
}

// headerOnlyFixtures have mathematically-zero AC content, so every AC nibble is float rounding
// noise sitting on a quantization tie; only the header bytes carry signal.
var headerOnlyFixtures = []string{"solid.png", "tiny.png"}

func isHeaderOnly(name string) bool {
	return slices.Contains(headerOnlyFixtures, name)
}

func loadGoldens() map[string]string {
	GinkgoHelper()
	data, err := os.ReadFile(filepath.Join(testdataDir, "golden.json"))
	Expect(err).ToNot(HaveOccurred())
	var golden map[string]string
	Expect(json.Unmarshal(data, &golden)).To(Succeed())
	Expect(golden).ToNot(BeEmpty())
	return golden
}

var _ = Describe("reference port", func() {
	It("reproduces every golden vector", func() {
		for name, want := range loadGoldens() {
			if isHeaderOnly(name) {
				continue // see the dedicated header-only spec below
			}
			w, h, rgba := loadFixture(name)
			got := base64.StdEncoding.EncodeToString(referenceEncode(w, h, rgba))
			Expect(got).To(Equal(want), "fixture %s", name)
		}
	})

	It("reproduces the well-conditioned header of the ill-conditioned fixtures", func() {
		for _, name := range headerOnlyFixtures {
			want, err := base64.StdEncoding.DecodeString(loadGoldens()[name])
			Expect(err).ToNot(HaveOccurred(), "fixture %s", name)
			w, h, rgba := loadFixture(name)
			Expect(referenceEncode(w, h, rgba)[:5]).To(Equal(want[:5]), "fixture %s header bytes", name)
		}
	})

	It("quantizes a uniform image's scales to zero", func() {
		w, h, rgba := loadFixture("solid.png")
		got := referenceEncode(w, h, rgba)
		header24 := int(got[0]) | int(got[1])<<8 | int(got[2])<<16
		header16 := int(got[3]) | int(got[4])<<8
		Expect((header24>>18)&31).To(Equal(0), "lScale")
		Expect((header16>>3)&63).To(Equal(0), "pScale")
		Expect((header16>>9)&63).To(Equal(0), "qScale")
	})

	It("produces 24 bytes for a square opaque image", func() {
		w, h, rgba := loadFixture("square.png")
		Expect(referenceEncode(w, h, rgba)).To(HaveLen(24))
	})

	It("produces 25 bytes when the image has alpha", func() {
		w, h, rgba := loadFixture("alpha.png")
		Expect(referenceEncode(w, h, rgba)).To(HaveLen(25))
	})
})

// referenceEncode is a literal port of evanw/thumbhash's rgbaToThumbHash (testdata/thumbhash.js).
// It is the differential oracle and the naive benchmark baseline; the shipped encoder is Encode.
func referenceEncode(w, h int, rgba []byte) []byte {
	var avgR, avgG, avgB, avgA float64
	for i, j := 0, 0; i < w*h; i, j = i+1, j+4 {
		alpha := float64(rgba[j+3]) / 255
		avgR += alpha / 255 * float64(rgba[j])
		avgG += alpha / 255 * float64(rgba[j+1])
		avgB += alpha / 255 * float64(rgba[j+2])
		avgA += alpha
	}
	if avgA > 0 {
		avgR /= avgA
		avgG /= avgA
		avgB /= avgA
	}

	hasAlpha := avgA < float64(w*h)
	lLimit := 7.0
	if hasAlpha {
		lLimit = 5.0
	}
	maxWH := float64(max(w, h))
	lx := max(1, int(math.Round(lLimit*float64(w)/maxWH)))
	ly := max(1, int(math.Round(lLimit*float64(h)/maxWH)))

	l := make([]float64, w*h)
	p := make([]float64, w*h)
	q := make([]float64, w*h)
	a := make([]float64, w*h)
	for i, j := 0, 0; i < w*h; i, j = i+1, j+4 {
		alpha := float64(rgba[j+3]) / 255
		r := avgR*(1-alpha) + alpha/255*float64(rgba[j])
		g := avgG*(1-alpha) + alpha/255*float64(rgba[j+1])
		b := avgB*(1-alpha) + alpha/255*float64(rgba[j+2])
		l[i] = (r + g + b) / 3
		p[i] = (r+g)/2 - b
		q[i] = r - g
		a[i] = alpha
	}

	encodeChannel := func(channel []float64, nx, ny int) (dc float64, ac []float64, scale float64) {
		fx := make([]float64, w)
		for cy := range ny {
			for cx := 0; cx*ny < nx*(ny-cy); cx++ {
				f := 0.0
				for x := range w {
					fx[x] = math.Cos(math.Pi / float64(w) * float64(cx) * (float64(x) + 0.5))
				}
				for y := range h {
					fy := math.Cos(math.Pi / float64(h) * float64(cy) * (float64(y) + 0.5))
					for x := range w {
						f += channel[x+y*w] * fx[x] * fy
					}
				}
				f /= float64(w * h)
				if cx > 0 || cy > 0 {
					ac = append(ac, f)
					scale = math.Max(scale, math.Abs(f))
				} else {
					dc = f
				}
			}
		}
		// A constant image leaves scale at 0; the reference then skips normalization entirely.
		if scale > 0 {
			for i := range ac {
				ac[i] = 0.5 + 0.5/scale*ac[i]
			}
		}
		return dc, ac, scale
	}

	lDC, lAC, lScale := encodeChannel(l, max(3, lx), max(3, ly))
	pDC, pAC, pScale := encodeChannel(p, 3, 3)
	qDC, qAC, qScale := encodeChannel(q, 3, 3)
	var aDC, aScale float64
	var aAC []float64
	if hasAlpha {
		aDC, aAC, aScale = encodeChannel(a, 5, 5)
	}

	isLandscape := 0
	if w > h {
		isLandscape = 1
	}
	alphaBit := 0
	if hasAlpha {
		alphaBit = 1
	}
	header24 := int(math.Round(63*lDC)) | int(math.Round(31.5+31.5*pDC))<<6 |
		int(math.Round(31.5+31.5*qDC))<<12 | int(math.Round(31*lScale))<<18 | alphaBit<<23
	lead := lx
	if isLandscape == 1 {
		lead = ly
	}
	header16 := lead | int(math.Round(63*pScale))<<3 | int(math.Round(63*qScale))<<9 | isLandscape<<15

	acs := [][]float64{lAC, pAC, qAC}
	acStart := 5
	if hasAlpha {
		acs = append(acs, aAC)
		acStart = 6
	}
	acCount := 0
	for _, ac := range acs {
		acCount += len(ac)
	}

	hash := make([]byte, acStart+(acCount+1)/2)
	hash[0] = byte(header24 & 255)
	hash[1] = byte((header24 >> 8) & 255)
	hash[2] = byte(header24 >> 16)
	hash[3] = byte(header16 & 255)
	hash[4] = byte(header16 >> 8)
	if hasAlpha {
		hash[5] = byte(int(math.Round(15*aDC)) | int(math.Round(15*aScale))<<4)
	}
	acIndex := 0
	for _, ac := range acs {
		for _, f := range ac {
			hash[acStart+(acIndex>>1)] |= byte(int(math.Round(15*f)) << ((acIndex & 1) << 2))
			acIndex++
		}
	}
	return hash
}
