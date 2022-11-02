package subsonic

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("helpers", func() {
	Describe("fakePath", func() {
		var mf model.MediaFile
		BeforeEach(func() {
			mf.AlbumArtist = "Brock Berrigan"
			mf.Album = "Point Pleasant"
			mf.Title = "Split Decision"
			mf.Suffix = "flac"
		})
		When("TrackNumber is not available", func() {
			It("does not add any number to the filename", func() {
				Expect(fakePath(mf)).To(Equal("Brock Berrigan/Point Pleasant/Split Decision.flac"))
			})
		})
		When("TrackNumber is available", func() {
			It("adds the trackNumber to the path", func() {
				mf.TrackNumber = 4
				Expect(fakePath(mf)).To(Equal("Brock Berrigan/Point Pleasant/04 - Split Decision.flac"))
			})
		})
	})

	Describe("mapSlashToDash", func() {
		It("maps / to _", func() {
			Expect(mapSlashToDash("AC/DC")).To(Equal("AC_DC"))
		})
	})
})
