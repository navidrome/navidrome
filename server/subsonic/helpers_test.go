package subsonic

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
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

	Describe("buildDiscTitles", func() {
		It("should return nil when album has no discs", func() {
			album := model.Album{}
			Expect(buildDiscSubtitles(context.Background(), album)).To(BeNil())
		})

		It("should return correct disc titles when album has discs with valid disc numbers", func() {
			album := model.Album{
				Discs: map[string]string{
					"1": "Disc 1",
					"2": "Disc 2",
				},
			}
			expected := responses.DiscTitles{
				{Disc: 1, Title: "Disc 1"},
				{Disc: 2, Title: "Disc 2"},
			}
			Expect(buildDiscSubtitles(context.Background(), album)).To(Equal(expected))
		})

		It("should skip discs with invalid disc numbers", func() {
			album := model.Album{
				Discs: map[string]string{
					"1":   "Disc 1",
					"two": "Disc 2",
				},
			}
			expected := responses.DiscTitles{
				{Disc: 1, Title: "Disc 1"},
			}
			Expect(buildDiscSubtitles(context.Background(), album)).To(Equal(expected))
		})
	})
})
