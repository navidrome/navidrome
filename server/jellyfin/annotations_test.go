package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Annotations", func() {
	var api *Router
	var ds *tests.MockDataStore
	// alice has access to library 1 only.
	ctxUser := func() context.Context {
		return request.WithUser(context.Background(), model.User{ID: "u1", UserName: "alice", Libraries: model.Libraries{{ID: 1, Name: "Music"}}})
	}

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		api = &Router{ds: ds}
	})

	Describe("markFavorite / unmarkFavorite", func() {
		It("stars a song and returns IsFavorite=true", func() {
			mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/s1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.IsFavorite).To(BeTrue())
			Expect(mfRepo.Data["s1"].Starred).To(BeTrue())
		})

		It("stars an album and returns IsFavorite=true", func() {
			albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: "a1", Name: "One", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/"+dto.EncodeID("a1"), nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID("a1"))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.IsFavorite).To(BeTrue())
			Expect(albumRepo.Data["a1"].Starred).To(BeTrue())
		})

		It("stars an artist without checking library access (artists span multiple libraries)", func() {
			artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
			w := httptest.NewRecorder()
			// alice only has access to library 1, but artists aren't gated per-library.
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/ar1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "ar1")
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.IsFavorite).To(BeTrue())
			Expect(artistRepo.Data["ar1"].Starred).To(BeTrue())
		})

		It("stars a visible playlist", func() {
			playlistRepo := ds.Playlist(context.Background()).(*tests.MockPlaylistRepo)
			playlistRepo.SetData(model.Playlists{{ID: "p1", Name: "Mix", OwnerID: "u1"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/"+dto.EncodeID("p1"), nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID("p1"))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(playlistRepo.Starred["p1"]).To(BeTrue())
		})

		It("unstars a song and returns IsFavorite=false", func() {
			mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song", LibraryID: 1, Annotations: model.Annotations{Starred: true}}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", "/Users/u1/FavoriteItems/s1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.unmarkFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.IsFavorite).To(BeFalse())
			Expect(mfRepo.Data["s1"].Starred).To(BeFalse())
		})

		It("returns 404 and does not star an album in a library the user can't access", func() {
			albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: "a1", Name: "One", LibraryID: 2}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/"+dto.EncodeID("a1"), nil).WithContext(ctxUser()) // only has access to library 1
			r = withChiURLParam(r, "itemId", dto.EncodeID("a1"))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(albumRepo.Data["a1"].Starred).To(BeFalse())
		})

		It("returns 404 and does not star a song in a library the user can't access", func() {
			mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song", LibraryID: 2}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/s1", nil).WithContext(ctxUser()) // only has access to library 1
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(mfRepo.Data["s1"].Starred).To(BeFalse())
		})

		It("returns 404 when the id doesn't match any entity", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/missing", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "missing")
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 500 (not 404) when a repository lookup fails for a reason other than not-found", func() {
			ds.Album(context.Background()).(*tests.MockAlbumRepo).SetError(true)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/x1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "x1")
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("setRating / removeRating", func() {
		It("maps a Jellyfin 0-10 rating to Navidrome's 0-5 scale", func() {
			mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/s1/Rating?Rating=8", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data["s1"].Rating).To(Equal(4))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.Rating).NotTo(BeNil())
			Expect(*d.Rating).To(Equal(8.0))
		})

		It("rates an album", func() {
			albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: "a1", Name: "One", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/"+dto.EncodeID("a1")+"/Rating?Rating=10", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID("a1"))
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(albumRepo.Data["a1"].Rating).To(Equal(5))
		})

		It("rates a visible playlist", func() {
			playlistRepo := ds.Playlist(context.Background()).(*tests.MockPlaylistRepo)
			playlistRepo.SetData(model.Playlists{{ID: "p1", Name: "Mix", OwnerID: "u1"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/"+dto.EncodeID("p1")+"/Rating?Rating=8", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID("p1"))
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(playlistRepo.Ratings["p1"]).To(Equal(4))
		})

		It("removes a rating", func() {
			mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song", LibraryID: 1, Annotations: model.Annotations{Rating: 4}}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", "/Users/u1/Items/s1/Rating", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.removeRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data["s1"].Rating).To(Equal(0))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.Rating).To(BeNil())
		})

		It("returns 404 and does not rate an album in a library the user can't access", func() {
			albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: "a1", Name: "One", LibraryID: 2}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/"+dto.EncodeID("a1")+"/Rating?Rating=10", nil).WithContext(ctxUser()) // only has access to library 1
			r = withChiURLParam(r, "itemId", dto.EncodeID("a1"))
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(albumRepo.Data["a1"].Rating).To(Equal(0))
		})

		It("rounds an odd rating to the nearest star instead of truncating", func() {
			mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/s1/Rating?Rating=9", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data["s1"].Rating).To(Equal(5))
		})

		It("stores the minimum star for Rating=1 instead of clearing the rating", func() {
			mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song", LibraryID: 1, Annotations: model.Annotations{Rating: 4}}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/s1/Rating?Rating=1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data["s1"].Rating).To(Equal(1))
		})

		It("accepts a fractional rating (UserItemDataDto.Rating is a double)", func() {
			mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/s1/Rating?Rating=7.5", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data["s1"].Rating).To(Equal(4))
		})

		It("clamps a Rating above 10 to Navidrome's max (5)", func() {
			mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/s1/Rating?Rating=100", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data["s1"].Rating).To(Equal(5))
		})

		It("clamps a negative Rating to Navidrome's min (0)", func() {
			mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/s1/Rating?Rating=-5", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data["s1"].Rating).To(Equal(0))
		})
	})
})
