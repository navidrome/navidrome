package subsonic

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Album Lists", func() {
	var ctx context.Context
	var router *Router
	var ds model.DataStore
	var w *httptest.ResponseRecorder

	childIDs := func(children []responses.Child) []string {
		return slice.Map(children, func(c responses.Child) string { return c.Id })
	}
	albumID3IDs := func(albums []responses.AlbumID3) []string {
		return slice.Map(albums, func(a responses.AlbumID3) string { return a.Id })
	}
	starredUser := model.User{ID: "test-user", UserName: "test-user"}

	// putTwelveAlbums stores al-01 (oldest) to al-12 (newest) in library 1.
	putTwelveAlbums := func() {
		GinkgoHelper()
		base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 1; i <= 12; i++ {
			putAlbums(ctx, ds, model.Album{ID: fmt.Sprintf("al-%02d", i), Name: fmt.Sprintf("Album %02d", i),
				CreatedAt: base.Add(time.Duration(i) * time.Hour)})
		}
	}

	// putLibraryContent stores artist ar-N, album al-N and rock song mf-N in each library N of 1-4,
	// all starred by test-user, who can read libraries 1-3 only.
	putLibraryContent := func() {
		GinkgoHelper()
		putUsers(ctx, ds, starredUser)
		libIDs := persistencetest.SetLibraries(ctx, ds, "/lib1", "/lib2", "/lib3", "/lib4")
		userCtx := request.WithUser(ctx, starredUser)
		rock := model.NewTag(model.TagGenre, "rock")
		for _, id := range libIDs {
			n := strconv.Itoa(id)
			Expect(ds.Tag().Add(ctx, id, rock)).To(Succeed())
			putAlbumArtists(ctx, ds, id, model.Artist{ID: "ar-" + n, Name: "Artist " + n})
			putAlbums(ctx, ds, model.Album{ID: "al-" + n, Name: "Album " + n, LibraryID: id})
			putTracks(ctx, ds, model.MediaFile{ID: "mf-" + n, Title: "Song " + n, LibraryID: id,
				Tags: model.Tags{model.TagGenre: {"rock"}}})
			Expect(ds.Artist().SetStar(userCtx, true, "ar-"+n)).To(Succeed())
			Expect(ds.Album().SetStar(userCtx, true, "al-"+n)).To(Succeed())
			Expect(ds.MediaFile().SetStar(userCtx, true, "mf-"+n)).To(Succeed())
		}
		Expect(ds.User().SetUserLibraries(ctx, starredUser.ID, libIDs[:3])).To(Succeed())
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		auth.Init(ds)
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		w = httptest.NewRecorder()
	})

	failingAlbumsRouter := func() *Router {
		return New(persistencetest.Overlay(ds, func(ds model.DataStore) model.DataStore { return failingAlbumGetAll{ds} }),
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	}

	Describe("GetAlbumList", func() {
		It("should return list of the type specified", func() {
			putTwelveAlbums()
			r := newGetRequest("type=newest", "offset=10", "size=20")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(childIDs(resp.AlbumList.Album)).To(Equal([]string{"al-02", "al-01"}))
			Expect(w.Header().Get("x-total-count")).To(Equal("12"))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()
			_, err := router.GetAlbumList(w, r)

			Expect(err).To(MatchError(req.ErrMissingParam))
		})

		It("should return error if call fails", func() {
			r := newGetRequest("type=newest")

			_, err := failingAlbumsRouter().GetAlbumList(w, r)

			Expect(err).To(MatchError(errSubsonic))
			var subErr subError
			errors.As(err, &subErr)
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		})

		Context("with musicFolderId parameter", func() {
			var user model.User

			BeforeEach(func() {
				putLibraryContent()
				user = model.User{
					ID: "test-user",
					Libraries: []model.Library{
						{ID: 1, Name: "Library 1"},
						{ID: 2, Name: "Library 2"},
						{ID: 3, Name: "Library 3"},
					},
				}
				ctx = request.WithUser(ctx, user)
			})

			It("should filter albums by specific library when musicFolderId is provided", func() {
				r := newGetRequest("type=newest", "musicFolderId=1")
				r = r.WithContext(ctx)

				resp, err := router.GetAlbumList(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(childIDs(resp.AlbumList.Album)).To(ConsistOf("al-1"))
			})

			It("should filter albums by multiple libraries when multiple musicFolderId are provided", func() {
				r := newGetRequest("type=newest", "musicFolderId=1", "musicFolderId=2")
				r = r.WithContext(ctx)

				resp, err := router.GetAlbumList(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(childIDs(resp.AlbumList.Album)).To(ConsistOf("al-1", "al-2"))
			})

			It("should return all accessible albums when no musicFolderId is provided", func() {
				r := newGetRequest("type=newest")
				r = r.WithContext(ctx)

				resp, err := router.GetAlbumList(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(childIDs(resp.AlbumList.Album)).To(ConsistOf("al-1", "al-2", "al-3"))
			})
		})
	})

	Describe("GetAlbumList2", func() {
		It("should return list of the type specified", func() {
			putTwelveAlbums()
			r := newGetRequest("type=newest", "offset=10", "size=20")
			resp, err := router.GetAlbumList2(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(albumID3IDs(resp.AlbumList2.Album)).To(Equal([]string{"al-02", "al-01"}))
			Expect(w.Header().Get("x-total-count")).To(Equal("12"))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()

			_, err := router.GetAlbumList2(w, r)

			Expect(err).To(MatchError(req.ErrMissingParam))
		})

		It("should return error if call fails", func() {
			r := newGetRequest("type=newest")

			_, err := failingAlbumsRouter().GetAlbumList2(w, r)

			Expect(err).To(MatchError(errSubsonic))
			var subErr subError
			errors.As(err, &subErr)
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		})

		Context("with musicFolderId parameter", func() {
			var user model.User

			BeforeEach(func() {
				putLibraryContent()
				user = model.User{
					ID: "test-user",
					Libraries: []model.Library{
						{ID: 1, Name: "Library 1"},
						{ID: 2, Name: "Library 2"},
						{ID: 3, Name: "Library 3"},
					},
				}
				ctx = request.WithUser(ctx, user)
			})

			It("should filter albums by specific library when musicFolderId is provided", func() {
				r := newGetRequest("type=newest", "musicFolderId=1")
				r = r.WithContext(ctx)

				resp, err := router.GetAlbumList2(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(albumID3IDs(resp.AlbumList2.Album)).To(ConsistOf("al-1"))
			})

			It("should filter albums by multiple libraries when multiple musicFolderId are provided", func() {
				r := newGetRequest("type=newest", "musicFolderId=1", "musicFolderId=2")
				r = r.WithContext(ctx)

				resp, err := router.GetAlbumList2(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(albumID3IDs(resp.AlbumList2.Album)).To(ConsistOf("al-1", "al-2"))
			})

			It("should return all accessible albums when no musicFolderId is provided", func() {
				r := newGetRequest("type=newest")
				r = r.WithContext(ctx)

				resp, err := router.GetAlbumList2(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(albumID3IDs(resp.AlbumList2.Album)).To(ConsistOf("al-1", "al-2", "al-3"))
			})
		})
	})

	Describe("GetRandomSongs", func() {
		It("should return random songs", func() {
			putTracks(ctx, ds,
				model.MediaFile{ID: "1", Title: "Song 1"},
				model.MediaFile{ID: "2", Title: "Song 2"},
			)
			r := newGetRequest("size=2")

			resp, err := router.GetRandomSongs(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(childIDs(resp.RandomSongs.Songs)).To(ConsistOf("1", "2"))
		})

		Context("with musicFolderId parameter", func() {
			var user model.User

			BeforeEach(func() {
				putLibraryContent()
				user = model.User{
					ID: "test-user",
					Libraries: []model.Library{
						{ID: 1, Name: "Library 1"},
						{ID: 2, Name: "Library 2"},
						{ID: 3, Name: "Library 3"},
					},
				}
				ctx = request.WithUser(ctx, user)
			})

			It("should filter songs by specific library when musicFolderId is provided", func() {
				r := newGetRequest("size=2", "musicFolderId=1")
				r = r.WithContext(ctx)

				resp, err := router.GetRandomSongs(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(childIDs(resp.RandomSongs.Songs)).To(ConsistOf("mf-1"))
			})

			It("should filter songs by multiple libraries when multiple musicFolderId are provided", func() {
				r := newGetRequest("size=2", "musicFolderId=1", "musicFolderId=2")
				r = r.WithContext(ctx)

				resp, err := router.GetRandomSongs(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(childIDs(resp.RandomSongs.Songs)).To(ConsistOf("mf-1", "mf-2"))
			})

			It("should return all accessible songs when no musicFolderId is provided", func() {
				r := newGetRequest("size=10")
				r = r.WithContext(ctx)

				resp, err := router.GetRandomSongs(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(childIDs(resp.RandomSongs.Songs)).To(ConsistOf("mf-1", "mf-2", "mf-3"))
			})
		})
	})

	Describe("GetSongsByGenre", func() {
		It("should return songs by genre", func() {
			rock := model.NewTag(model.TagGenre, "rock")
			jazz := model.NewTag(model.TagGenre, "jazz")
			Expect(ds.Tag().Add(ctx, model.DefaultLibraryID, rock, jazz)).To(Succeed())
			putTracks(ctx, ds,
				model.MediaFile{ID: "1", Title: "Song 1", Tags: model.Tags{model.TagGenre: {"rock"}}},
				model.MediaFile{ID: "2", Title: "Song 2", Tags: model.Tags{model.TagGenre: {"rock"}}},
				model.MediaFile{ID: "3", Title: "Song 3", Tags: model.Tags{model.TagGenre: {"jazz"}}},
			)
			r := newGetRequest("count=2", "genre=rock")

			resp, err := router.GetSongsByGenre(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(childIDs(resp.SongsByGenre.Songs)).To(ConsistOf("1", "2"))
		})

		Context("with musicFolderId parameter", func() {
			var user model.User

			BeforeEach(func() {
				putLibraryContent()
				user = model.User{
					ID: "test-user",
					Libraries: []model.Library{
						{ID: 1, Name: "Library 1"},
						{ID: 2, Name: "Library 2"},
						{ID: 3, Name: "Library 3"},
					},
				}
				ctx = request.WithUser(ctx, user)
			})

			It("should filter songs by specific library when musicFolderId is provided", func() {
				r := newGetRequest("count=2", "genre=rock", "musicFolderId=1")
				r = r.WithContext(ctx)

				resp, err := router.GetSongsByGenre(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(childIDs(resp.SongsByGenre.Songs)).To(ConsistOf("mf-1"))
			})

			It("should filter songs by multiple libraries when multiple musicFolderId are provided", func() {
				r := newGetRequest("count=2", "genre=rock", "musicFolderId=1", "musicFolderId=2")
				r = r.WithContext(ctx)

				resp, err := router.GetSongsByGenre(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(childIDs(resp.SongsByGenre.Songs)).To(ConsistOf("mf-1", "mf-2"))
			})

			It("should return all accessible songs when no musicFolderId is provided", func() {
				r := newGetRequest("count=10", "genre=rock")
				r = r.WithContext(ctx)

				resp, err := router.GetSongsByGenre(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(childIDs(resp.SongsByGenre.Songs)).To(ConsistOf("mf-1", "mf-2", "mf-3"))
			})
		})
	})

	// putStarredAndPlain stores ar-1, al-1 and mf-1 starred by test-user, and unstarred ar-2, al-2 and mf-2.
	putStarredAndPlain := func() {
		GinkgoHelper()
		putUsers(ctx, ds, starredUser)
		putAlbumArtists(ctx, ds, model.DefaultLibraryID,
			model.Artist{ID: "ar-1", Name: "Artist 1"}, model.Artist{ID: "ar-2", Name: "Artist 2"})
		putAlbums(ctx, ds, model.Album{ID: "al-1", Name: "Album 1"}, model.Album{ID: "al-2", Name: "Album 2"})
		putTracks(ctx, ds, model.MediaFile{ID: "mf-1", Title: "Song 1"}, model.MediaFile{ID: "mf-2", Title: "Song 2"})
		userCtx := request.WithUser(ctx, starredUser)
		Expect(ds.Artist().SetStar(userCtx, true, "ar-1")).To(Succeed())
		Expect(ds.Album().SetStar(userCtx, true, "al-1")).To(Succeed())
		Expect(ds.MediaFile().SetStar(userCtx, true, "mf-1")).To(Succeed())
	}

	Describe("GetStarred", func() {
		It("should return starred items", func() {
			putStarredAndPlain()
			r := newGetRequest().WithContext(request.WithUser(ctx, starredUser))

			resp, err := router.GetStarred(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(slice.Map(resp.Starred.Artist, func(a responses.Artist) string { return a.Id })).To(ConsistOf("ar-1"))
			Expect(childIDs(resp.Starred.Album)).To(ConsistOf("al-1"))
			Expect(childIDs(resp.Starred.Song)).To(ConsistOf("mf-1"))
		})

		Context("with musicFolderId parameter", func() {
			var user model.User

			BeforeEach(func() {
				putLibraryContent()
				user = model.User{
					ID: "test-user",
					Libraries: []model.Library{
						{ID: 1, Name: "Library 1"},
						{ID: 2, Name: "Library 2"},
						{ID: 3, Name: "Library 3"},
					},
				}
				ctx = request.WithUser(ctx, user)
			})

			It("should filter starred items by specific library when musicFolderId is provided", func() {
				r := newGetRequest("musicFolderId=1")
				r = r.WithContext(ctx)

				resp, err := router.GetStarred(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(slice.Map(resp.Starred.Artist, func(a responses.Artist) string { return a.Id })).To(ConsistOf("ar-1"))
				Expect(childIDs(resp.Starred.Album)).To(ConsistOf("al-1"))
				Expect(childIDs(resp.Starred.Song)).To(ConsistOf("mf-1"))
			})
		})
	})

	Describe("GetStarred2", func() {
		It("should return starred items in ID3 format", func() {
			putStarredAndPlain()
			r := newGetRequest().WithContext(request.WithUser(ctx, starredUser))

			resp, err := router.GetStarred2(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(slice.Map(resp.Starred2.Artist, func(a responses.ArtistID3) string { return a.Id })).To(ConsistOf("ar-1"))
			Expect(albumID3IDs(resp.Starred2.Album)).To(ConsistOf("al-1"))
			Expect(childIDs(resp.Starred2.Song)).To(ConsistOf("mf-1"))
		})

		Context("with musicFolderId parameter", func() {
			var user model.User

			BeforeEach(func() {
				putLibraryContent()
				user = model.User{
					ID: "test-user",
					Libraries: []model.Library{
						{ID: 1, Name: "Library 1"},
						{ID: 2, Name: "Library 2"},
						{ID: 3, Name: "Library 3"},
					},
				}
				ctx = request.WithUser(ctx, user)
			})

			It("should filter starred items by specific library when musicFolderId is provided", func() {
				r := newGetRequest("musicFolderId=1")
				r = r.WithContext(ctx)

				resp, err := router.GetStarred2(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(slice.Map(resp.Starred2.Artist, func(a responses.ArtistID3) string { return a.Id })).To(ConsistOf("ar-1"))
				Expect(albumID3IDs(resp.Starred2.Album)).To(ConsistOf("al-1"))
				Expect(childIDs(resp.Starred2.Song)).To(ConsistOf("mf-1"))
			})
		})
	})
})

type failingAlbumGetAll struct{ model.DataStore }

func (d failingAlbumGetAll) Album() model.AlbumRepository {
	return failingAlbumGetAllRepo{d.DataStore.Album()}
}

type failingAlbumGetAllRepo struct{ model.AlbumRepository }

func (failingAlbumGetAllRepo) GetAll(context.Context, ...model.QueryOptions) (model.Albums, error) {
	return nil, errors.New("db error")
}
