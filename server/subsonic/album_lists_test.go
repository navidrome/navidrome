package subsonic

import (
	"context"
	"errors"
	"net/http/httptest"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/req"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Album Lists", func() {
	var router *Router
	var ds model.DataStore
	var mockRepo *tests.MockAlbumRepo
	var w *httptest.ResponseRecorder
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		auth.Init(ds)
		mockRepo = ds.Album(ctx).(*tests.MockAlbumRepo)
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		w = httptest.NewRecorder()
	})

	Describe("GetAlbumList", func() {
		It("should return list of the type specified", func() {
			r := newGetRequest("type=newest", "offset=10", "size=20")
			mockRepo.SetData(model.Albums{
				{ID: "1"}, {ID: "2"},
			})
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList.Album[0].Id).To(Equal("1"))
			Expect(resp.AlbumList.Album[1].Id).To(Equal("2"))
			Expect(w.Header().Get("x-total-count")).To(Equal("2"))
			Expect(mockRepo.Options.Offset).To(Equal(10))
			Expect(mockRepo.Options.Max).To(Equal(20))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()
			_, err := router.GetAlbumList(w, r)

			Expect(err).To(MatchError(req.ErrMissingParam))
		})

		It("should return error if call fails", func() {
			mockRepo.SetError(true)
			r := newGetRequest("type=newest")

			_, err := router.GetAlbumList(w, r)

			Expect(err).To(MatchError(errSubsonic))
			var subErr subError
			errors.As(err, &subErr)
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		})

		Context("with musicFolderId parameter", func() {
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

			It("should filter albums by specific library when musicFolderId is provided", func() {
				r := newGetRequest("type=newest", "musicFolderId=1")
				r = r.WithContext(ctx)
				mockRepo.SetData(model.Albums{
					{ID: "1"}, {ID: "2"},
				})

				resp, err := router.GetAlbumList(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.AlbumList.Album).To(HaveLen(2))
				// Verify that library filter was applied
				query, args, _ := mockRepo.Options.Filters.ToSql()
				Expect(query).To(ContainSubstring("library_id IN (?)"))
				Expect(args).To(ContainElement(1))
			})

			It("should filter albums by multiple libraries when multiple musicFolderId are provided", func() {
				r := newGetRequest("type=newest", "musicFolderId=1", "musicFolderId=2")
				r = r.WithContext(ctx)
				mockRepo.SetData(model.Albums{
					{ID: "1"}, {ID: "2"},
				})

				resp, err := router.GetAlbumList(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.AlbumList.Album).To(HaveLen(2))
				// Verify that library filter was applied
				query, args, _ := mockRepo.Options.Filters.ToSql()
				Expect(query).To(ContainSubstring("library_id IN (?,?)"))
				Expect(args).To(ContainElements(1, 2))
			})

			It("should return all accessible albums when no musicFolderId is provided", func() {
				r := newGetRequest("type=newest")
				r = r.WithContext(ctx)
				mockRepo.SetData(model.Albums{
					{ID: "1"}, {ID: "2"},
				})

				resp, err := router.GetAlbumList(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.AlbumList.Album).To(HaveLen(2))
				// Verify that library filter was applied
				query, args, _ := mockRepo.Options.Filters.ToSql()
				Expect(query).To(ContainSubstring("library_id IN (?,?,?)"))
				Expect(args).To(ContainElements(1, 2, 3))
			})
		})
	})

	Describe("GetAlbumList2", func() {
		It("should return list of the type specified", func() {
			r := newGetRequest("type=newest", "offset=10", "size=20")
			mockRepo.SetData(model.Albums{
				{ID: "1"}, {ID: "2"},
			})
			resp, err := router.GetAlbumList2(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList2.Album[0].Id).To(Equal("1"))
			Expect(resp.AlbumList2.Album[1].Id).To(Equal("2"))
			Expect(w.Header().Get("x-total-count")).To(Equal("2"))
			Expect(mockRepo.Options.Offset).To(Equal(10))
			Expect(mockRepo.Options.Max).To(Equal(20))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()

			_, err := router.GetAlbumList2(w, r)

			Expect(err).To(MatchError(req.ErrMissingParam))
		})

		It("should return error if call fails", func() {
			mockRepo.SetError(true)
			r := newGetRequest("type=newest")

			_, err := router.GetAlbumList2(w, r)

			Expect(err).To(MatchError(errSubsonic))
			var subErr subError
			errors.As(err, &subErr)
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		})

		Context("with musicFolderId parameter", func() {
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

			It("should filter albums by specific library when musicFolderId is provided", func() {
				r := newGetRequest("type=newest", "musicFolderId=1")
				r = r.WithContext(ctx)
				mockRepo.SetData(model.Albums{
					{ID: "1"}, {ID: "2"},
				})

				resp, err := router.GetAlbumList2(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.AlbumList2.Album).To(HaveLen(2))
				// Verify that library filter was applied
				Expect(mockRepo.Options.Filters).ToNot(BeNil())
			})

			It("should filter albums by multiple libraries when multiple musicFolderId are provided", func() {
				r := newGetRequest("type=newest", "musicFolderId=1", "musicFolderId=2")
				r = r.WithContext(ctx)
				mockRepo.SetData(model.Albums{
					{ID: "1"}, {ID: "2"},
				})

				resp, err := router.GetAlbumList2(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.AlbumList2.Album).To(HaveLen(2))
				// Verify that library filter was applied
				Expect(mockRepo.Options.Filters).ToNot(BeNil())
			})

			It("should return all accessible albums when no musicFolderId is provided", func() {
				r := newGetRequest("type=newest")
				r = r.WithContext(ctx)
				mockRepo.SetData(model.Albums{
					{ID: "1"}, {ID: "2"},
				})

				resp, err := router.GetAlbumList2(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.AlbumList2.Album).To(HaveLen(2))
			})
		})
	})

	Describe("GetRandomSongs", func() {
		var mockMediaFileRepo *tests.MockMediaFileRepo

		BeforeEach(func() {
			mockMediaFileRepo = ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
		})

		It("should return random songs", func() {
			mockMediaFileRepo.SetData(model.MediaFiles{
				{ID: "1", Title: "Song 1"},
				{ID: "2", Title: "Song 2"},
			})
			r := newGetRequest("size=2")

			resp, err := router.GetRandomSongs(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.RandomSongs.Songs).To(HaveLen(2))
		})

		Context("with musicFolderId parameter", func() {
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

			It("should filter songs by specific library when musicFolderId is provided", func() {
				mockMediaFileRepo.SetData(model.MediaFiles{
					{ID: "1", Title: "Song 1"},
					{ID: "2", Title: "Song 2"},
				})
				r := newGetRequest("size=2", "musicFolderId=1")
				r = r.WithContext(ctx)

				resp, err := router.GetRandomSongs(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.RandomSongs.Songs).To(HaveLen(2))
				// Verify that library filter was applied
				query, args, _ := mockMediaFileRepo.Options.Filters.ToSql()
				Expect(query).To(ContainSubstring("library_id IN (?)"))
				Expect(args).To(ContainElement(1))
			})

			It("should filter songs by multiple libraries when multiple musicFolderId are provided", func() {
				mockMediaFileRepo.SetData(model.MediaFiles{
					{ID: "1", Title: "Song 1"},
					{ID: "2", Title: "Song 2"},
				})
				r := newGetRequest("size=2", "musicFolderId=1", "musicFolderId=2")
				r = r.WithContext(ctx)

				resp, err := router.GetRandomSongs(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.RandomSongs.Songs).To(HaveLen(2))
				// Verify that library filter was applied
				query, args, _ := mockMediaFileRepo.Options.Filters.ToSql()
				Expect(query).To(ContainSubstring("library_id IN (?,?)"))
				Expect(args).To(ContainElements(1, 2))
			})

			It("should return all accessible songs when no musicFolderId is provided", func() {
				mockMediaFileRepo.SetData(model.MediaFiles{
					{ID: "1", Title: "Song 1"},
					{ID: "2", Title: "Song 2"},
				})
				r := newGetRequest("size=2")
				r = r.WithContext(ctx)

				resp, err := router.GetRandomSongs(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.RandomSongs.Songs).To(HaveLen(2))
				// Verify that library filter was applied
				query, args, _ := mockMediaFileRepo.Options.Filters.ToSql()
				Expect(query).To(ContainSubstring("library_id IN (?,?,?)"))
				Expect(args).To(ContainElements(1, 2, 3))
			})
		})
	})

	Describe("GetSongsByGenre", func() {
		var mockMediaFileRepo *tests.MockMediaFileRepo

		BeforeEach(func() {
			mockMediaFileRepo = ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
		})

		It("should return songs by genre", func() {
			mockMediaFileRepo.SetData(model.MediaFiles{
				{ID: "1", Title: "Song 1"},
				{ID: "2", Title: "Song 2"},
			})
			r := newGetRequest("count=2", "genre=rock")

			resp, err := router.GetSongsByGenre(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SongsByGenre.Songs).To(HaveLen(2))
		})

		Context("with musicFolderId parameter", func() {
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

			It("should filter songs by specific library when musicFolderId is provided", func() {
				mockMediaFileRepo.SetData(model.MediaFiles{
					{ID: "1", Title: "Song 1"},
					{ID: "2", Title: "Song 2"},
				})
				r := newGetRequest("count=2", "genre=rock", "musicFolderId=1")
				r = r.WithContext(ctx)

				resp, err := router.GetSongsByGenre(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.SongsByGenre.Songs).To(HaveLen(2))
				// Verify that library filter was applied
				query, args, _ := mockMediaFileRepo.Options.Filters.ToSql()
				Expect(query).To(ContainSubstring("library_id IN (?)"))
				Expect(args).To(ContainElement(1))
			})

			It("should filter songs by multiple libraries when multiple musicFolderId are provided", func() {
				mockMediaFileRepo.SetData(model.MediaFiles{
					{ID: "1", Title: "Song 1"},
					{ID: "2", Title: "Song 2"},
				})
				r := newGetRequest("count=2", "genre=rock", "musicFolderId=1", "musicFolderId=2")
				r = r.WithContext(ctx)

				resp, err := router.GetSongsByGenre(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.SongsByGenre.Songs).To(HaveLen(2))
				// Verify that library filter was applied
				query, args, _ := mockMediaFileRepo.Options.Filters.ToSql()
				Expect(query).To(ContainSubstring("library_id IN (?,?)"))
				Expect(args).To(ContainElements(1, 2))
			})

			It("should return all accessible songs when no musicFolderId is provided", func() {
				mockMediaFileRepo.SetData(model.MediaFiles{
					{ID: "1", Title: "Song 1"},
					{ID: "2", Title: "Song 2"},
				})
				r := newGetRequest("count=2", "genre=rock")
				r = r.WithContext(ctx)

				resp, err := router.GetSongsByGenre(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.SongsByGenre.Songs).To(HaveLen(2))
				// Verify that library filter was applied
				query, args, _ := mockMediaFileRepo.Options.Filters.ToSql()
				Expect(query).To(ContainSubstring("library_id IN (?,?,?)"))
				Expect(args).To(ContainElements(1, 2, 3))
			})
		})
	})

	Describe("GetStarred", func() {
		var mockArtistRepo *tests.MockArtistRepo
		var mockAlbumRepo *tests.MockAlbumRepo
		var mockMediaFileRepo *tests.MockMediaFileRepo

		BeforeEach(func() {
			mockArtistRepo = ds.Artist(ctx).(*tests.MockArtistRepo)
			mockAlbumRepo = ds.Album(ctx).(*tests.MockAlbumRepo)
			mockMediaFileRepo = ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
		})

		It("should return starred items", func() {
			mockArtistRepo.SetData(model.Artists{{ID: "1", Name: "Artist 1"}})
			mockAlbumRepo.SetData(model.Albums{{ID: "1", Name: "Album 1"}})
			mockMediaFileRepo.SetData(model.MediaFiles{{ID: "1", Title: "Song 1"}})
			r := newGetRequest()

			resp, err := router.GetStarred(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Starred.Artist).To(HaveLen(1))
			Expect(resp.Starred.Album).To(HaveLen(1))
			Expect(resp.Starred.Song).To(HaveLen(1))
		})

		Context("with musicFolderId parameter", func() {
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

			It("should filter starred items by specific library when musicFolderId is provided", func() {
				mockArtistRepo.SetData(model.Artists{{ID: "1", Name: "Artist 1"}})
				mockAlbumRepo.SetData(model.Albums{{ID: "1", Name: "Album 1"}})
				mockMediaFileRepo.SetData(model.MediaFiles{{ID: "1", Title: "Song 1"}})
				r := newGetRequest("musicFolderId=1")
				r = r.WithContext(ctx)

				resp, err := router.GetStarred(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Starred.Artist).To(HaveLen(1))
				Expect(resp.Starred.Album).To(HaveLen(1))
				Expect(resp.Starred.Song).To(HaveLen(1))
				// Verify that library filter was applied to all types
				artistQuery, artistArgs, _ := mockArtistRepo.Options.Filters.ToSql()
				Expect(artistQuery).To(ContainSubstring("library_id IN (?)"))
				Expect(artistArgs).To(ContainElement(1))
			})
		})
	})

	Describe("GetStarred2", func() {
		var mockArtistRepo *tests.MockArtistRepo
		var mockAlbumRepo *tests.MockAlbumRepo
		var mockMediaFileRepo *tests.MockMediaFileRepo

		BeforeEach(func() {
			mockArtistRepo = ds.Artist(ctx).(*tests.MockArtistRepo)
			mockAlbumRepo = ds.Album(ctx).(*tests.MockAlbumRepo)
			mockMediaFileRepo = ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
		})

		It("should return starred items in ID3 format", func() {
			mockArtistRepo.SetData(model.Artists{{ID: "1", Name: "Artist 1"}})
			mockAlbumRepo.SetData(model.Albums{{ID: "1", Name: "Album 1"}})
			mockMediaFileRepo.SetData(model.MediaFiles{{ID: "1", Title: "Song 1"}})
			r := newGetRequest()

			resp, err := router.GetStarred2(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Starred2.Artist).To(HaveLen(1))
			Expect(resp.Starred2.Album).To(HaveLen(1))
			Expect(resp.Starred2.Song).To(HaveLen(1))
		})

		Context("with musicFolderId parameter", func() {
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

			It("should filter starred items by specific library when musicFolderId is provided", func() {
				mockArtistRepo.SetData(model.Artists{{ID: "1", Name: "Artist 1"}})
				mockAlbumRepo.SetData(model.Albums{{ID: "1", Name: "Album 1"}})
				mockMediaFileRepo.SetData(model.MediaFiles{{ID: "1", Title: "Song 1"}})
				r := newGetRequest("musicFolderId=1")
				r = r.WithContext(ctx)

				resp, err := router.GetStarred2(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Starred2.Artist).To(HaveLen(1))
				Expect(resp.Starred2.Album).To(HaveLen(1))
				Expect(resp.Starred2.Song).To(HaveLen(1))
				// Verify that library filter was applied to all types
				artistQuery, artistArgs, _ := mockArtistRepo.Options.Filters.ToSql()
				Expect(artistQuery).To(ContainSubstring("library_id IN (?)"))
				Expect(artistArgs).To(ContainElement(1))
			})
		})
	})
})
