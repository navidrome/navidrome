package e2e

import (
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Radio art is uploaded-image-only, with no fallback. Uploads are real files on disk, so they
// serve back byte-for-byte; a radio with no upload settles absent.
var _ = Describe("Radio artwork resolution", func() {
	BeforeEach(func() {
		setupResolutionHarness()
	})

	When("a radio has an uploaded image", func() {
		// <DataFolder>/
		// └── artwork/
		//     └── radio/
		//         └── rd-1_logo.jpg   ← matched by UploadedImagePath()
		It("returns the uploaded image bytes", func() {
			writeUploadedImage(consts.EntityRadio, "rd-1_logo.jpg", pngBytes("radio-logo"))
			rd := model.Radio{ID: "rd-1", Name: "Test Radio", StreamUrl: "https://example.com/stream", UploadedImage: "rd-1_logo.jpg"}
			Expect(rds.Radio(rctx).Put(&rd)).To(Succeed())

			ia := acquire(model.KindRadioArtwork, rd.ID)
			Expect(ia.Source).To(Equal("upload"))
			Expect(serveBytes(model.NewArtworkID(model.KindRadioArtwork, rd.ID, nil))).To(Equal(pngBytes("radio-logo")))
		})
	})

	When("a radio has no uploaded image", func() {
		// (no files on disk — the resolver has no sources to fall back to)
		It("settles absent", func() {
			rd := model.Radio{ID: "rd-2", Name: "Bare Radio", StreamUrl: "https://example.com/stream"}
			Expect(rds.Radio(rctx).Put(&rd)).To(Succeed())

			ia := acquire(model.KindRadioArtwork, rd.ID)
			Expect(ia.Hash).To(BeEmpty())
			Expect(serveErr(model.NewArtworkID(model.KindRadioArtwork, rd.ID, nil))).To(MatchError(artwork.ErrUnavailable))
		})
	})
})
