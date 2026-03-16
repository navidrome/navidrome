package model_test

import (
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artist", func() {
	Describe("UploadedImagePath", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.DataFolder = "/data"
		})

		It("returns empty string when no image uploaded", func() {
			a := model.Artist{ID: "ar-1"}
			Expect(a.UploadedImagePath()).To(BeEmpty())
		})

		It("returns full path when image is set", func() {
			a := model.Artist{ID: "ar-1", UploadedImage: "ar-1_test.jpg"}
			Expect(a.UploadedImagePath()).To(Equal(filepath.Join("/data", "artwork", "artist", "ar-1_test.jpg")))
		})
	})
})
