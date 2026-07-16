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

var _ = Describe("Artist", func() {
	Describe("UploadedImagePath", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.DataFolder = conf.NewDir("/data")
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

var _ = Describe("Artist.ArtworkUpdatedAt", func() {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	later := base.Add(24 * time.Hour)

	It("handles nil timestamps", func() {
		Expect(model.Artist{}.ArtworkUpdatedAt()).To(Equal(time.Time{}))
	})
	It("returns UpdatedAt", func() {
		Expect(model.Artist{UpdatedAt: &later, ExternalInfoUpdatedAt: &base}.ArtworkUpdatedAt()).To(Equal(later))
	})
	It("ignores ExternalInfoUpdatedAt (agent TTL refreshes bump it without an image change)", func() {
		Expect(model.Artist{UpdatedAt: &base, ExternalInfoUpdatedAt: &later}.ArtworkUpdatedAt()).To(Equal(base))
	})
})
