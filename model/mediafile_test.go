package model_test

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaFile", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DevFastAccessCoverArt = false
	})
	Describe(".CoverArtId()", func() {
		It("returns its own id if it HasCoverArt", func() {
			mf := model.MediaFile{ID: "111", AlbumID: "1", HasCoverArt: true}
			id := mf.CoverArtID()
			Expect(id.Kind).To(Equal(model.KindMediaFileArtwork))
			Expect(id.ID).To(Equal(mf.ID))
		})
		It("returns its album id if HasCoverArt is false", func() {
			mf := model.MediaFile{ID: "111", AlbumID: "1", HasCoverArt: false}
			id := mf.CoverArtID()
			Expect(id.Kind).To(Equal(model.KindAlbumArtwork))
			Expect(id.ID).To(Equal(mf.AlbumID))
		})
		It("returns its album id if DevFastAccessCoverArt is enabled", func() {
			conf.Server.DevFastAccessCoverArt = true
			mf := model.MediaFile{ID: "111", AlbumID: "1", HasCoverArt: true}
			id := mf.CoverArtID()
			Expect(id.Kind).To(Equal(model.KindAlbumArtwork))
			Expect(id.ID).To(Equal(mf.AlbumID))
		})
	})
})
