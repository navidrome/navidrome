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

	DescribeTable("isClientInList",
		func(list, client string, expected bool) {
			Expect(isClientInList(list, client)).To(Equal(expected))
		},
		Entry("returns false when clientList is empty", "", "some-client", false),
		Entry("returns false when client is empty", "client1,client2", "", false),
		Entry("returns false when both are empty", "", "", false),
		Entry("returns true when client matches single entry", "my-client", "my-client", true),
		Entry("returns true when client matches first in list", "client1,client2,client3", "client1", true),
		Entry("returns true when client matches middle in list", "client1,client2,client3", "client2", true),
		Entry("returns true when client matches last in list", "client1,client2,client3", "client3", true),
		Entry("returns false when client does not match", "client1,client2", "client3", false),
		Entry("trims whitespace from client list entries", "client1, client2 , client3", "client2", true),
		Entry("does not trim the client parameter", "client1,client2", " client1", false),
	)

	Describe("childFromMediaFile", func() {
		var mf model.MediaFile
		var ctx context.Context

		BeforeEach(func() {
			mf = model.MediaFile{
				ID:          "mf-1",
				Title:       "Test Song",
				Album:       "Test Album",
				AlbumID:     "album-1",
				Artist:      "Test Artist",
				ArtistID:    "artist-1",
				Year:        2023,
				Genre:       "Rock",
				TrackNumber: 5,
				Duration:    180.5,
				Size:        5000000,
				Suffix:      "mp3",
				BitRate:     320,
			}
			ctx = context.Background()
		})

		Context("with minimal client", func() {
			BeforeEach(func() {
				conf.Server.Subsonic.MinimalClients = "minimal-client"
				player := model.Player{Client: "minimal-client"}
				ctx = request.WithPlayer(ctx, player)
			})

			It("returns only basic fields", func() {
				child := childFromMediaFile(ctx, mf)
				Expect(child.Id).To(Equal("mf-1"))
				Expect(child.Title).To(Equal("Test Song"))
				Expect(child.IsDir).To(BeFalse())

				// These should not be set
				Expect(child.Album).To(BeEmpty())
				Expect(child.Artist).To(BeEmpty())
				Expect(child.Parent).To(BeEmpty())
				Expect(child.Year).To(BeZero())
				Expect(child.Genre).To(BeEmpty())
				Expect(child.Track).To(BeZero())
				Expect(child.Duration).To(BeZero())
				Expect(child.Size).To(BeZero())
				Expect(child.Suffix).To(BeEmpty())
				Expect(child.BitRate).To(BeZero())
				Expect(child.CoverArt).To(BeEmpty())
				Expect(child.ContentType).To(BeEmpty())
				Expect(child.Path).To(BeEmpty())
			})

			It("does not include OpenSubsonic extension", func() {
				child := childFromMediaFile(ctx, mf)
				Expect(child.OpenSubsonicChild).To(BeNil())
			})
		})

		Context("with non-minimal client", func() {
			BeforeEach(func() {
				conf.Server.Subsonic.MinimalClients = "minimal-client"
				player := model.Player{Client: "regular-client"}
				ctx = request.WithPlayer(ctx, player)
			})

			It("returns all fields", func() {
				child := childFromMediaFile(ctx, mf)
				Expect(child.Id).To(Equal("mf-1"))
				Expect(child.Title).To(Equal("Test Song"))
				Expect(child.IsDir).To(BeFalse())
				Expect(child.Album).To(Equal("Test Album"))
				Expect(child.Artist).To(Equal("Test Artist"))
				Expect(child.Parent).To(Equal("album-1"))
				Expect(child.Year).To(Equal(int32(2023)))
				Expect(child.Genre).To(Equal("Rock"))
				Expect(child.Track).To(Equal(int32(5)))
				Expect(child.Duration).To(Equal(int32(180)))
				Expect(child.Size).To(Equal(int64(5000000)))
				Expect(child.Suffix).To(Equal("mp3"))
				Expect(child.BitRate).To(Equal(int32(320)))
			})
		})

		Context("when minimal clients list is empty", func() {
			BeforeEach(func() {
				conf.Server.Subsonic.MinimalClients = ""
				player := model.Player{Client: "any-client"}
				ctx = request.WithPlayer(ctx, player)
			})

			It("returns all fields", func() {
				child := childFromMediaFile(ctx, mf)
				Expect(child.Album).To(Equal("Test Album"))
				Expect(child.Artist).To(Equal("Test Artist"))
			})
		})

		Context("when no player in context", func() {
			It("returns all fields", func() {
				child := childFromMediaFile(ctx, mf)
				Expect(child.Album).To(Equal("Test Album"))
				Expect(child.Artist).To(Equal("Test Artist"))
			})
		})
	})

	Describe("osChildFromMediaFile", func() {
		var mf model.MediaFile
		var ctx context.Context

		BeforeEach(func() {
			mf = model.MediaFile{
				ID:      "mf-1",
				Title:   "Test Song",
				Artist:  "Test Artist",
				Comment: "Test Comment",
			}
			ctx = context.Background()
		})

		Context("with minimal client", func() {
			BeforeEach(func() {
				conf.Server.Subsonic.MinimalClients = "minimal-client"
				player := model.Player{Client: "minimal-client"}
				ctx = request.WithPlayer(ctx, player)
			})

			It("returns nil", func() {
				osChild := osChildFromMediaFile(ctx, mf)
				Expect(osChild).To(BeNil())
			})
		})

		Context("with non-minimal client", func() {
			BeforeEach(func() {
				conf.Server.Subsonic.MinimalClients = "minimal-client"
				player := model.Player{Client: "regular-client"}
				ctx = request.WithPlayer(ctx, player)
			})

			It("returns OpenSubsonic child fields", func() {
				osChild := osChildFromMediaFile(ctx, mf)
				Expect(osChild).ToNot(BeNil())
				Expect(osChild.Comment).To(Equal("Test Comment"))
			})
		})

		Context("when minimal clients list is empty", func() {
			BeforeEach(func() {
				conf.Server.Subsonic.MinimalClients = ""
				player := model.Player{Client: "any-client"}
				ctx = request.WithPlayer(ctx, player)
			})

			It("returns OpenSubsonic child fields", func() {
				osChild := osChildFromMediaFile(ctx, mf)
				Expect(osChild).ToNot(BeNil())
			})
		})

		Context("when no player in context", func() {
			It("returns OpenSubsonic child fields", func() {
				osChild := osChildFromMediaFile(ctx, mf)
				Expect(osChild).ToNot(BeNil())
			})
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

	Describe("AverageRating in responses", func() {
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
			conf.Server.Subsonic.EnableAverageRating = true
		})

		Describe("childFromMediaFile", func() {
			It("includes averageRating when set", func() {
				mf := model.MediaFile{
					ID:    "mf-avg-1",
					Title: "Test Song",
					Annotations: model.Annotations{
						AverageRating: 4.5,
					},
				}
				child := childFromMediaFile(ctx, mf)
				Expect(child.AverageRating).To(Equal(4.5))
			})

			It("returns 0 for averageRating when not set", func() {
				mf := model.MediaFile{
					ID:    "mf-avg-2",
					Title: "Test Song No Rating",
				}
				child := childFromMediaFile(ctx, mf)
				Expect(child.AverageRating).To(Equal(0.0))
			})
		})

		Describe("childFromAlbum", func() {
			It("includes averageRating when set", func() {
				al := model.Album{
					ID:   "al-avg-1",
					Name: "Test Album",
					Annotations: model.Annotations{
						AverageRating: 3.75,
					},
				}
				child := childFromAlbum(ctx, al)
				Expect(child.AverageRating).To(Equal(3.75))
			})

			It("returns 0 for averageRating when not set", func() {
				al := model.Album{
					ID:   "al-avg-2",
					Name: "Test Album No Rating",
				}
				child := childFromAlbum(ctx, al)
				Expect(child.AverageRating).To(Equal(0.0))
			})
		})

		Describe("toArtist", func() {
			It("includes averageRating when set", func() {
				conf.Server.Subsonic.EnableAverageRating = true
				r := httptest.NewRequest("GET", "/test", nil)
				a := model.Artist{
					ID:   "ar-avg-1",
					Name: "Test Artist",
					Annotations: model.Annotations{
						AverageRating: 5.0,
					},
				}
				artist := toArtist(r, a)
				Expect(artist.AverageRating).To(Equal(5.0))
			})
		})

		Describe("toArtistID3", func() {
			It("includes averageRating when set", func() {
				conf.Server.Subsonic.EnableAverageRating = true
				r := httptest.NewRequest("GET", "/test", nil)
				a := model.Artist{
					ID:   "ar-avg-2",
					Name: "Test Artist ID3",
					Annotations: model.Annotations{
						AverageRating: 2.5,
					},
				}
				artist := toArtistID3(r, a)
				Expect(artist.AverageRating).To(Equal(2.5))
			})
		})

		Describe("EnableAverageRating config", func() {
			It("excludes averageRating when disabled", func() {
				conf.Server.Subsonic.EnableAverageRating = false

				mf := model.MediaFile{
					ID:    "mf-cfg-1",
					Title: "Test Song",
					Annotations: model.Annotations{
						AverageRating: 4.5,
					},
				}
				child := childFromMediaFile(ctx, mf)
				Expect(child.AverageRating).To(Equal(0.0))

				al := model.Album{
					ID:   "al-cfg-1",
					Name: "Test Album",
					Annotations: model.Annotations{
						AverageRating: 3.75,
					},
				}
				albumChild := childFromAlbum(ctx, al)
				Expect(albumChild.AverageRating).To(Equal(0.0))

				r := httptest.NewRequest("GET", "/test", nil)
				a := model.Artist{
					ID:   "ar-cfg-1",
					Name: "Test Artist",
					Annotations: model.Annotations{
						AverageRating: 5.0,
					},
				}
				artist := toArtist(r, a)
				Expect(artist.AverageRating).To(Equal(0.0))

				artistID3 := toArtistID3(r, a)
				Expect(artistID3.AverageRating).To(Equal(0.0))
			})
		})
	})
})
