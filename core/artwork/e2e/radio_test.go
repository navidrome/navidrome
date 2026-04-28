package artworke2e_test

import (
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Radio artwork resolution", func() {
	BeforeEach(func() {
		setupHarness()
	})

	When("a radio has an uploaded image", func() {
		// <DataFolder>/
		// └── artwork/
		//     └── radio/
		//         └── rd-1_logo.jpg   ← matched by UploadedImagePath()
		It("returns the uploaded image bytes", func() {
			writeUploadedImage(consts.EntityRadio, "rd-1_logo.jpg", imageBytes("radio-logo"))

			rd := model.Radio{ID: "rd-1", Name: "Test Radio", StreamUrl: "https://example.com/stream", UploadedImage: "rd-1_logo.jpg"}
			Expect(ds.Radio(ctx).Put(&rd)).To(Succeed())

			artID := model.NewArtworkID(model.KindRadioArtwork, rd.ID, nil)
			Expect(readArtwork(artID)).To(Equal(imageBytes("radio-logo")))
		})
	})

	When("a radio has no uploaded image", func() {
		// (no files on disk — reader has no sources to fall back to)
		It("returns ErrUnavailable", func() {
			rd := model.Radio{ID: "rd-2", Name: "Bare Radio", StreamUrl: "https://example.com/stream"}
			Expect(ds.Radio(ctx).Put(&rd)).To(Succeed())

			artID := model.NewArtworkID(model.KindRadioArtwork, rd.ID, nil)
			_, err := readArtworkOrErr(artID)
			Expect(err).To(HaveOccurred())
		})
	})
})
