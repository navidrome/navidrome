package artwork

import (
	"image"
	"image/color"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("makeThumbnail", func() {
	// Both encoders read Pix directly and thumbhash needs straight alpha, so emitting NRGBA is
	// what keeps either of them from allocating a converted copy of the thumbnail.
	It("emits NRGBA when it downscales", func() {
		src := image.NewRGBA(image.Rect(0, 0, 400, 300))
		Expect(makeThumbnail(src, 100)).To(BeAssignableToTypeOf(&image.NRGBA{}))
	})

	It("fits the longest side to maxSize and keeps the aspect ratio", func() {
		src := image.NewRGBA(image.Rect(0, 0, 400, 300))
		b := makeThumbnail(src, 100).Bounds()
		Expect(b.Dx()).To(Equal(100))
		Expect(b.Dy()).To(Equal(75))
	})

	It("never upscales an image already within bounds", func() {
		src := image.NewRGBA(image.Rect(0, 0, 40, 30))
		Expect(makeThumbnail(src, 100).Bounds()).To(Equal(src.Bounds()))
	})

	// A fully transparent pixel's colour cannot survive any resample, since the scaler works in
	// premultiplied space and multiplying by zero is not invertible. Partial alpha is the case
	// that distinguishes straight storage from premultiplied.
	It("keeps partly transparent colour straight rather than premultiplied", func() {
		src := image.NewNRGBA(image.Rect(0, 0, 400, 400))
		for y := range 400 {
			for x := range 400 {
				src.SetNRGBA(x, y, color.NRGBA{R: 200, G: 40, B: 90, A: 128})
			}
		}
		thumb, ok := makeThumbnail(src, 100).(*image.NRGBA)
		Expect(ok).To(BeTrue())
		// Premultiplied storage would have halved this to ~100.
		Expect(thumb.Pix[0]).To(BeNumerically("==", 200))
		Expect(thumb.Pix[3]).To(BeNumerically("==", 128))
	})
})
