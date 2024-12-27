package subsonic

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
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
		When("TrackNumber and DiscNumber are available", func() {
			It("adds the trackNumber to the path", func() {
				mf.TrackNumber = 4
				mf.DiscNumber = 1
				Expect(fakePath(mf)).To(Equal("Brock Berrigan/Point Pleasant/01-04 - Split Decision.flac"))
			})
		})
	})

	Describe("sanitizeSlashes", func() {
		It("maps / to _", func() {
			Expect(sanitizeSlashes("AC/DC")).To(Equal("AC_DC"))
		})
	})

	Describe("sortName", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
		})
		When("PreferSortTags is false", func() {
			BeforeEach(func() {
				conf.Server.PreferSortTags = false
			})
			It("returns the order name even if sort name is provided", func() {
				Expect(sortName("Sort Album Name", "Order Album Name")).To(Equal("Order Album Name"))
			})
			It("returns the order name if sort name is empty", func() {
				Expect(sortName("", "Order Album Name")).To(Equal("Order Album Name"))
			})
		})
		When("PreferSortTags is true", func() {
			BeforeEach(func() {
				conf.Server.PreferSortTags = true
			})
			It("returns the sort name if provided", func() {
				Expect(sortName("Sort Album Name", "Order Album Name")).To(Equal("Sort Album Name"))
			})

			It("returns the order name if sort name is empty", func() {
				Expect(sortName("", "Order Album Name")).To(Equal("Order Album Name"))
			})
		})
		It("returns an empty string if both sort name and order name are empty", func() {
			Expect(sortName("", "")).To(Equal(""))
		})
	})

	Describe("buildDiscTitles", func() {
		It("should return nil when album has no discs", func() {
			album := model.Album{}
			Expect(buildDiscSubtitles(album)).To(BeNil())
		})

		It("should return correct disc titles when album has discs with valid disc numbers", func() {
			album := model.Album{
				Discs: map[int]string{
					1: "Disc 1",
					2: "Disc 2",
				},
			}
			expected := []responses.DiscTitle{
				{Disc: 1, Title: "Disc 1"},
				{Disc: 2, Title: "Disc 2"},
			}
			Expect(buildDiscSubtitles(album)).To(Equal(expected))
		})
	})

	DescribeTable("toItemDate",
		func(date string, expected responses.ItemDate) {
			Expect(toItemDate(date)).To(Equal(expected))
		},
		Entry("1994-02-04", "1994-02-04", responses.ItemDate{Year: 1994, Month: 2, Day: 4}),
		Entry("1994-02", "1994-02", responses.ItemDate{Year: 1994, Month: 2}),
		Entry("1994", "1994", responses.ItemDate{Year: 1994}),
		Entry("19940201", "", responses.ItemDate{}),
		Entry("", "", responses.ItemDate{}),
	)

	DescribeTable("mapExplicitStatus",
		func(explicitStatus string, expected string) {
			Expect(mapExplicitStatus(explicitStatus)).To(Equal(expected))
		},
		Entry("returns \"clean\" when the db value is \"c\"", "c", "clean"),
		Entry("returns \"explicit\" when the db value is \"e\"", "e", "explicit"),
		Entry("returns an empty string when the db value is \"\"", "", ""),
		Entry("returns an empty string when there are unexpected values on the db", "abc", ""))
})
