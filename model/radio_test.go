package model_test

import (
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Radio", func() {
	Describe("CoverArtID", func() {
		It("returns a radio artwork ID", func() {
			now := time.Now()
			r := model.Radio{ID: "rd-1", UpdatedAt: now}
			artID := r.CoverArtID()
			Expect(artID.Kind).To(Equal(model.KindRadioArtwork))
			Expect(artID.ID).To(Equal("rd-1"))
			Expect(artID.LastUpdate).To(Equal(now))
		})
	})

	Describe("UploadedImagePath", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.DataFolder = "/data"
		})

		It("returns empty string when no image uploaded", func() {
			r := model.Radio{ID: "rd-1"}
			Expect(r.UploadedImagePath()).To(BeEmpty())
		})

		It("returns full path when image is set", func() {
			r := model.Radio{ID: "rd-1", UploadedImage: "rd-1_test.jpg"}
			Expect(r.UploadedImagePath()).To(Equal(filepath.Join("/data", "artwork", "radio", "rd-1_test.jpg")))
		})
	})
})
