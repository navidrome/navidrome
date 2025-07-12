package subsonic

import (
	"context"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("helpers", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

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

		It("should return nil when album has only one disc without title", func() {
			album := model.Album{
				Discs: map[int]string{
					1: "",
				},
			}
			Expect(buildDiscSubtitles(album)).To(BeNil())
		})

		It("should return the disc title for a single disc", func() {
			album := model.Album{
				Discs: map[int]string{
					1: "Special Edition",
				},
			}
			Expect(buildDiscSubtitles(album)).To(Equal([]responses.DiscTitle{{Disc: 1, Title: "Special Edition"}}))
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

	Describe("getArtistAlbumCount", func() {
		artist := model.Artist{
			Stats: map[model.Role]model.ArtistStats{
				model.RoleAlbumArtist: {
					AlbumCount: 3,
				},
				model.RoleMainCredit: {
					AlbumCount: 4,
				},
			},
		}

		It("Handles album count without artist participations", func() {
			conf.Server.Subsonic.ArtistParticipations = false
			result := getArtistAlbumCount(&artist)
			Expect(result).To(Equal(int32(3)))
		})

		It("Handles album count without with participations", func() {
			conf.Server.Subsonic.ArtistParticipations = true
			result := getArtistAlbumCount(&artist)
			Expect(result).To(Equal(int32(4)))
		})
	})

	Describe("selectedMusicFolderIds", func() {
		var user model.User
		var ctx context.Context

		BeforeEach(func() {
			user = model.User{
				ID: "test-user",
				Libraries: []model.Library{
					{ID: 1, Name: "Library 1"},
					{ID: 2, Name: "Library 2"},
					{ID: 3, Name: "Library 3"},
				},
			}
			ctx = request.WithUser(context.Background(), user)
		})

		Context("when musicFolderId parameter is provided", func() {
			It("should return the specified musicFolderId values", func() {
				r := httptest.NewRequest("GET", "/test?musicFolderId=1&musicFolderId=3", nil)
				r = r.WithContext(ctx)

				ids, err := selectedMusicFolderIds(r, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(ids).To(Equal([]int{1, 3}))
			})

			It("should ignore invalid musicFolderId parameter values", func() {
				r := httptest.NewRequest("GET", "/test?musicFolderId=invalid&musicFolderId=2", nil)
				r = r.WithContext(ctx)

				ids, err := selectedMusicFolderIds(r, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(ids).To(Equal([]int{2})) // Only valid ID is returned
			})

			It("should return error when any library ID is not accessible", func() {
				r := httptest.NewRequest("GET", "/test?musicFolderId=1&musicFolderId=5&musicFolderId=2&musicFolderId=99", nil)
				r = r.WithContext(ctx)

				ids, err := selectedMusicFolderIds(r, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Library 5 not found or not accessible"))
				Expect(ids).To(BeNil())
			})
		})

		Context("when musicFolderId parameter is not provided", func() {
			Context("and required is false", func() {
				It("should return all user's library IDs", func() {
					r := httptest.NewRequest("GET", "/test", nil)
					r = r.WithContext(ctx)

					ids, err := selectedMusicFolderIds(r, false)
					Expect(err).ToNot(HaveOccurred())
					Expect(ids).To(Equal([]int{1, 2, 3}))
				})

				It("should return empty slice when user has no libraries", func() {
					userWithoutLibs := model.User{ID: "no-libs-user", Libraries: []model.Library{}}
					ctxWithoutLibs := request.WithUser(context.Background(), userWithoutLibs)
					r := httptest.NewRequest("GET", "/test", nil)
					r = r.WithContext(ctxWithoutLibs)

					ids, err := selectedMusicFolderIds(r, false)
					Expect(err).ToNot(HaveOccurred())
					Expect(ids).To(Equal([]int{}))
				})
			})

			Context("and required is true", func() {
				It("should return ErrMissingParam error", func() {
					r := httptest.NewRequest("GET", "/test", nil)
					r = r.WithContext(ctx)

					ids, err := selectedMusicFolderIds(r, true)
					Expect(err).To(MatchError(req.ErrMissingParam))
					Expect(ids).To(BeNil())
				})
			})
		})

		Context("when musicFolderId parameter is empty", func() {
			It("should return all user's library IDs even when empty parameter is provided", func() {
				r := httptest.NewRequest("GET", "/test?musicFolderId=", nil)
				r = r.WithContext(ctx)

				ids, err := selectedMusicFolderIds(r, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(ids).To(Equal([]int{1, 2, 3}))
			})
		})

		Context("when all musicFolderId parameters are invalid", func() {
			It("should return all user libraries when all musicFolderId parameters are invalid", func() {
				r := httptest.NewRequest("GET", "/test?musicFolderId=invalid&musicFolderId=notanumber", nil)
				r = r.WithContext(ctx)

				ids, err := selectedMusicFolderIds(r, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(ids).To(Equal([]int{1, 2, 3})) // Falls back to all user libraries
			})
		})
	})
})
