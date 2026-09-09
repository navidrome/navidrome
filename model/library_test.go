package model_test

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Library", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.PID.Album = "album_global"
		conf.Server.PID.Track = "track_global"
	})

	Describe("EffectivePIDAlbum", func() {
		It("returns the global config when the override is empty", func() {
			lib := model.Library{}
			Expect(lib.EffectivePIDAlbum()).To(Equal("album_global"))
		})

		It("returns the override when it is set", func() {
			lib := model.Library{PIDAlbum: "folder"}
			Expect(lib.EffectivePIDAlbum()).To(Equal("folder"))
		})
	})

	Describe("EffectivePIDTrack", func() {
		It("returns the global config when the override is empty", func() {
			lib := model.Library{}
			Expect(lib.EffectivePIDTrack()).To(Equal("track_global"))
		})

		It("returns the override when it is set", func() {
			lib := model.Library{PIDTrack: "title"}
			Expect(lib.EffectivePIDTrack()).To(Equal("title"))
		})
	})
})
