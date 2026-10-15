package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Annotations", func() {
	var api *Router
	var ds *tests.MockDataStore
	var broker *fakeEventBroker
	// alice has access to library 1 only.
	ctxUser := func() context.Context {
		return request.WithUser(GinkgoT().Context(), model.User{ID: testID("u1"), UserName: "alice", Libraries: model.Libraries{{ID: 1, Name: "Music"}}})
	}

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		broker = &fakeEventBroker{}
		api = &Router{ds: ds, broker: broker}
	})

	songWith := func(ann model.Annotations) *tests.MockMediaFileRepo {
		mfRepo := ds.MediaFile().(*tests.MockMediaFileRepo)
		mfRepo.SetData(model.MediaFiles{{ID: testID("s1"), Title: "Song", LibraryID: 1, Annotations: ann}})
		return mfRepo
	}
	songReq := func(method, target, body string, params ...string) *http.Request {
		r := httptest.NewRequest(method, target, strings.NewReader(body)).WithContext(ctxUser())
		return withChiURLParam(r, append([]string{"itemId", dto.EncodeID(testID("s1"))}, params...)...)
	}

	Describe("markFavorite / unmarkFavorite", func() {
		It("stars a song and returns IsFavorite=true", func() {
			mfRepo := ds.MediaFile().(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: testID("s1"), Title: "Song", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/s1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("s1")))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.IsFavorite).To(BeTrue())
			Expect(mfRepo.Data[testID("s1")].Starred).To(BeTrue())
		})

		It("stars an album and returns IsFavorite=true", func() {
			albumRepo := ds.Album().(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: testID("a1"), Name: "One", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/"+dto.EncodeID(testID("a1")), nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("a1")))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.IsFavorite).To(BeTrue())
			Expect(albumRepo.Data[testID("a1")].Starred).To(BeTrue())
		})

		It("stars an artist without checking library access (artists span multiple libraries)", func() {
			artistRepo := ds.Artist().(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: testID("ar1"), Name: "Artist"}})
			w := httptest.NewRecorder()
			// alice only has access to library 1, but artists aren't gated per-library.
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/ar1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("ar1")))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.IsFavorite).To(BeTrue())
			Expect(artistRepo.Data[testID("ar1")].Starred).To(BeTrue())
		})

		It("stars a visible playlist", func() {
			playlistRepo := ds.Playlist().(*tests.MockPlaylistRepo)
			playlistRepo.SetData(model.Playlists{{ID: testID("p1"), Name: "Mix", OwnerID: testID("u1")}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/"+dto.EncodeID(testID("p1")), nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("p1")))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(playlistRepo.Starred[testID("p1")]).To(BeTrue())
		})

		It("unstars a song and returns IsFavorite=false", func() {
			mfRepo := ds.MediaFile().(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: testID("s1"), Title: "Song", LibraryID: 1, Annotations: model.Annotations{Starred: true}}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", "/Users/u1/FavoriteItems/s1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("s1")))
			invoke(api.unmarkFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.IsFavorite).To(BeFalse())
			Expect(mfRepo.Data[testID("s1")].Starred).To(BeFalse())
		})

		It("returns 404 and does not star an album in a library the user can't access", func() {
			albumRepo := ds.Album().(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: testID("a1"), Name: "One", LibraryID: 2}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/"+dto.EncodeID(testID("a1")), nil).WithContext(ctxUser()) // only has access to library 1
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("a1")))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(albumRepo.Data[testID("a1")].Starred).To(BeFalse())
		})

		It("returns 404 and does not star a song in a library the user can't access", func() {
			mfRepo := ds.MediaFile().(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: testID("s1"), Title: "Song", LibraryID: 2}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/s1", nil).WithContext(ctxUser()) // only has access to library 1
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("s1")))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(mfRepo.Data[testID("s1")].Starred).To(BeFalse())
		})

		It("returns 404 when the id doesn't match any entity", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/missing", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("missing")))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 500 (not 404) when a repository lookup fails for a reason other than not-found", func() {
			ds.Album().(*tests.MockAlbumRepo).SetError(true)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/x1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("x1")))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})

		It("emits a refreshResource event when starring a song", func() {
			mfRepo := ds.MediaFile().(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: testID("s1"), Title: "Song", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/s1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("s1")))
			invoke(api.markFavorite, w, r)
			Expect(broker.Events).To(HaveLen(1))
			Expect(broker.Events[0].Data(broker.Events[0])).To(Equal(`{"song":["` + testID("s1") + `"]}`))
		})

		It("emits a refreshResource event when starring an album", func() {
			albumRepo := ds.Album().(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: testID("a1"), Name: "One", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/"+dto.EncodeID(testID("a1")), nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("a1")))
			invoke(api.markFavorite, w, r)
			Expect(broker.Events).To(HaveLen(1))
			Expect(broker.Events[0].Data(broker.Events[0])).To(Equal(`{"album":["` + testID("a1") + `"]}`))
		})

		It("does not emit an event when the item is not accessible", func() {
			albumRepo := ds.Album().(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: testID("a1"), Name: "One", LibraryID: 2}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/FavoriteItems/"+dto.EncodeID(testID("a1")), nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("a1")))
			invoke(api.markFavorite, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(broker.Events).To(BeEmpty())
		})
	})

	Describe("setRating / removeRating", func() {
		It("maps a Jellyfin 0-10 rating to Navidrome's 0-5 scale", func() {
			mfRepo := ds.MediaFile().(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: testID("s1"), Title: "Song", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/s1/Rating?Rating=8", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("s1")))
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(4))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.Rating).NotTo(BeNil())
			Expect(*d.Rating).To(Equal(8.0))
		})

		It("rates an album", func() {
			albumRepo := ds.Album().(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: testID("a1"), Name: "One", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/"+dto.EncodeID(testID("a1"))+"/Rating?Rating=10", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("a1")))
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(albumRepo.Data[testID("a1")].Rating).To(Equal(5))
		})

		It("rates a visible playlist", func() {
			playlistRepo := ds.Playlist().(*tests.MockPlaylistRepo)
			playlistRepo.SetData(model.Playlists{{ID: testID("p1"), Name: "Mix", OwnerID: testID("u1")}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/"+dto.EncodeID(testID("p1"))+"/Rating?Rating=8", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("p1")))
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(playlistRepo.Ratings[testID("p1")]).To(Equal(4))
		})

		It("removes a rating", func() {
			mfRepo := ds.MediaFile().(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: testID("s1"), Title: "Song", LibraryID: 1, Annotations: model.Annotations{Rating: 4}}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", "/Users/u1/Items/s1/Rating", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("s1")))
			invoke(api.removeRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(0))
			var d dto.UserItemDataDto
			Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
			Expect(d.Rating).To(BeNil())
		})

		It("returns 404 and does not rate an album in a library the user can't access", func() {
			albumRepo := ds.Album().(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: testID("a1"), Name: "One", LibraryID: 2}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/"+dto.EncodeID(testID("a1"))+"/Rating?Rating=10", nil).WithContext(ctxUser()) // only has access to library 1
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("a1")))
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(albumRepo.Data[testID("a1")].Rating).To(Equal(0))
		})

		It("rounds an odd rating to the nearest star instead of truncating", func() {
			mfRepo := ds.MediaFile().(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: testID("s1"), Title: "Song", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/s1/Rating?Rating=9", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("s1")))
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(5))
		})

		It("stores the minimum star for Rating=1 instead of clearing the rating", func() {
			mfRepo := ds.MediaFile().(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: testID("s1"), Title: "Song", LibraryID: 1, Annotations: model.Annotations{Rating: 4}}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/s1/Rating?Rating=1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("s1")))
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(1))
		})

		It("accepts a fractional rating (UserItemDataDto.Rating is a double)", func() {
			mfRepo := ds.MediaFile().(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: testID("s1"), Title: "Song", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/u1/Items/s1/Rating?Rating=7.5", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID(testID("s1")))
			invoke(api.setRating, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(4))
		})

		It("emits a refreshResource event when rating a song", func() {
			songWith(model.Annotations{})
			invoke(api.setRating, httptest.NewRecorder(), songReq("POST", "/Users/u1/Items/s1/Rating?Rating=8", ""))
			Expect(broker.Events).To(HaveLen(1))
			Expect(broker.Events[0].Data(broker.Events[0])).To(Equal(`{"song":["` + testID("s1") + `"]}`))
		})

		Context("Jellyfin likes semantics", func() {
			It("stores likes=true as 5 stars and reports Rating 10 / Likes true", func() {
				mfRepo := songWith(model.Annotations{})
				w := httptest.NewRecorder()
				invoke(api.setRating, w, songReq("POST", "/UserItems/s1/Rating?likes=true", ""))
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(5))
				d := parseUserData(w)
				Expect(*d.Rating).To(Equal(10.0))
				Expect(*d.Likes).To(BeTrue())
			})

			It("stores likes=false as 1 star (Jellyfin's dislike is Rating 1), not a clear", func() {
				mfRepo := songWith(model.Annotations{Rating: 4})
				w := httptest.NewRecorder()
				invoke(api.setRating, w, songReq("POST", "/UserItems/s1/Rating?Likes=False", ""))
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(1))
				d := parseUserData(w)
				Expect(*d.Rating).To(Equal(2.0))
				Expect(*d.Likes).To(BeFalse())
			})

			It("prefers likes over the numeric rating extension when both are sent", func() {
				mfRepo := songWith(model.Annotations{})
				w := httptest.NewRecorder()
				invoke(api.setRating, w, songReq("POST", "/UserItems/s1/Rating?likes=false&rating=10", ""))
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(1))
			})

			It("still clears on an explicit rating=0 from numeric clients", func() {
				mfRepo := songWith(model.Annotations{Rating: 4})
				w := httptest.NewRecorder()
				invoke(api.setRating, w, songReq("POST", "/UserItems/s1/Rating?rating=0", ""))
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(0))
				Expect(parseUserData(w).Rating).To(BeNil())
			})

			It("keeps the favorite and play state and reports them in the response", func() {
				playDate := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
				mfRepo := songWith(model.Annotations{Starred: true, PlayCount: 3, PlayDate: &playDate})
				w := httptest.NewRecorder()
				invoke(api.setRating, w, songReq("POST", "/UserItems/s1/Rating?likes=true", ""))
				Expect(w.Code).To(Equal(http.StatusOK))
				song := mfRepo.Data[testID("s1")]
				Expect(song.Starred).To(BeTrue())
				Expect(song.PlayCount).To(Equal(int64(3)))
				Expect(song.PlayDate).To(Equal(&playDate))
				d := parseUserData(w)
				Expect(d.IsFavorite).To(BeTrue())
				Expect(d.PlayCount).To(Equal(3))
				Expect(d.Played).To(BeTrue())
			})

			DescribeTable("rejects a missing or malformed value without touching the stored rating",
				func(query string) {
					mfRepo := songWith(model.Annotations{Rating: 4})
					w := httptest.NewRecorder()
					invoke(api.setRating, w, songReq("POST", "/UserItems/s1/Rating"+query, ""))
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(4))
					Expect(broker.Events).To(BeEmpty())
				},
				Entry("no params (Jellyfin would clear)", ""),
				Entry("unrelated params only", "?userId=x&foo=1"),
				Entry("empty likes", "?likes="),
				Entry("non-boolean likes", "?likes=maybe"),
				Entry("non-boolean likes even with a valid rating", "?likes=maybe&rating=8"),
				Entry("empty rating", "?rating="),
				Entry("non-numeric rating", "?rating=abc"),
				Entry("NaN rating", "?rating=NaN"),
				Entry("infinite rating", "?rating=Inf"),
				Entry("rating above 10", "?Rating=100"),
				Entry("negative rating", "?Rating=-5"),
			)
		})

		Context("authorization", func() {
			DescribeTable("refuses a userId naming another user and leaves the rating",
				func(method, target string, params ...string) {
					handler := api.setRating
					if method == "DELETE" {
						handler = api.removeRating
					}
					mfRepo := songWith(model.Annotations{Rating: 4})
					w := httptest.NewRecorder()
					invoke(handler, w, songReq(method, target, "", params...))
					Expect(w.Code).To(Equal(http.StatusForbidden))
					Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(4))
					Expect(broker.Events).To(BeEmpty())
				},
				Entry("path userId", "POST", "/Users/u2/Items/s1/Rating?likes=true", "userId", dto.EncodeID(testID("u2"))),
				Entry("query userId on the /UserItems form", "POST", "/UserItems/s1/Rating?likes=true&userId="+dto.EncodeID(testID("u2"))),
				Entry("clearing", "DELETE", "/Users/u2/Items/s1/Rating", "userId", dto.EncodeID(testID("u2"))),
			)

			It("accepts the caller's own userId", func() {
				mfRepo := songWith(model.Annotations{})
				w := httptest.NewRecorder()
				invoke(api.setRating, w, songReq("POST", "/Users/u1/Items/s1/Rating?likes=true", "", "userId", dto.EncodeID(testID("u1"))))
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(5))
			})
		})
	})

	Describe("updateUserItemData", func() {
		post := func(body string, params ...string) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			invoke(api.updateUserItemData, w, songReq("POST", "/UserItems/s1/UserData", body, params...))
			return w
		}

		It("maps a 0-10 Rating to stars and answers with the full UserData", func() {
			mfRepo := songWith(model.Annotations{Starred: true, PlayCount: 2})
			w := post(`{"Rating":8}`)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(4))
			d := parseUserData(w)
			Expect(*d.Rating).To(Equal(8.0))
			Expect(*d.Likes).To(BeTrue())
			Expect(d.IsFavorite).To(BeTrue())
			Expect(d.PlayCount).To(Equal(2))
			Expect(d.ItemId).To(Equal(dto.EncodeID(testID("s1"))))
			Expect(broker.Events).To(HaveLen(1))
		})

		DescribeTable("stores the personal rating",
			func(body string, before, expected int) {
				mfRepo := songWith(model.Annotations{Rating: before})
				Expect(post(body).Code).To(Equal(http.StatusOK))
				Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(expected))
			},
			Entry("Likes true", `{"Likes":true}`, 0, 5),
			Entry("Likes false (Rating 1)", `{"Likes":false}`, 4, 1),
			Entry("Rating wins over Likes, as Jellyfin applies Likes first", `{"Likes":true,"Rating":2}`, 0, 1),
			Entry("fractional Rating rounds to the nearest star", `{"Rating":7.5}`, 0, 4),
			Entry("small nonzero Rating floors at one star", `{"Rating":0.4}`, 0, 1),
			Entry("Rating 0 clears", `{"Rating":0}`, 4, 0),
		)

		DescribeTable("leaves the rating untouched when Rating and Likes are omitted or null",
			func(body string, wantEvents int) {
				mfRepo := songWith(model.Annotations{Rating: 4})
				w := post(body)
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(4))
				Expect(*parseUserData(w).Rating).To(Equal(8.0))
				Expect(broker.Events).To(HaveLen(wantEvents))
			},
			Entry("empty object writes nothing", `{}`, 0),
			Entry("explicit nulls write nothing", `{"Rating":null,"Likes":null,"IsFavorite":null}`, 0),
			Entry("favorite only", `{"IsFavorite":true}`, 1),
		)

		It("sets and unsets the favorite without touching the rating", func() {
			mfRepo := songWith(model.Annotations{Rating: 4})
			Expect(post(`{"IsFavorite":true}`).Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data[testID("s1")].Starred).To(BeTrue())
			Expect(post(`{"IsFavorite":false}`).Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data[testID("s1")].Starred).To(BeFalse())
			Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(4))
		})

		It("ignores play-state fields, which only change through playback reporting", func() {
			mfRepo := songWith(model.Annotations{PlayCount: 2})
			w := post(`{"Played":true,"PlayCount":99,"LastPlayedDate":"2026-01-01T00:00:00Z","PlaybackPositionTicks":5}`)
			Expect(w.Code).To(Equal(http.StatusOK))
			song := mfRepo.Data[testID("s1")]
			Expect(song.PlayCount).To(Equal(int64(2)))
			Expect(song.PlayDate).To(BeNil())
			Expect(broker.Events).To(BeEmpty())
		})

		It("leaves state unchanged when a client echoes back the UserData it read", func() {
			mfRepo := songWith(model.Annotations{Rating: 3, Starred: true, PlayCount: 1})
			echo, err := json.Marshal(dto.UserData(mfRepo.Data[testID("s1")].Annotations, testID("s1")))
			Expect(err).ToNot(HaveOccurred())
			Expect(post(string(echo)).Code).To(Equal(http.StatusOK))
			song := mfRepo.Data[testID("s1")]
			Expect(song.Rating).To(Equal(3))
			Expect(song.Starred).To(BeTrue())
			Expect(song.PlayCount).To(Equal(int64(1)))
		})

		DescribeTable("rejects an invalid body without writing anything",
			func(body string) {
				mfRepo := songWith(model.Annotations{Rating: 4})
				Expect(post(body).Code).To(Equal(http.StatusBadRequest))
				song := mfRepo.Data[testID("s1")]
				Expect(song.Rating).To(Equal(4))
				Expect(song.Starred).To(BeFalse())
				Expect(broker.Events).To(BeEmpty())
			},
			Entry("empty body", ""),
			Entry("malformed JSON", `{"Rating":`),
			Entry("Rating above 10, even with a valid IsFavorite", `{"IsFavorite":true,"Rating":10.5}`),
			Entry("negative Rating", `{"IsFavorite":true,"Rating":-1}`),
			Entry("Rating as a string", `{"Rating":"8"}`),
			Entry("Likes as a string", `{"Likes":"yes"}`),
			Entry("IsFavorite as a number", `{"IsFavorite":1}`),
			Entry("JSON null", `null`),
			Entry("JSON array", `[]`),
			Entry("trailing garbage after the object", `{"Rating":8} x`),
			Entry("a second JSON value after the object", `{"Rating":8}{"Rating":2}`),
			Entry("a second object after whitespace", "{\"Rating\":8}\n{}"),
		)

		It("accepts trailing whitespace after the object", func() {
			mfRepo := songWith(model.Annotations{})
			Expect(post("{\"Rating\":8}\r\n\t ").Code).To(Equal(http.StatusOK))
			Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(4))
		})

		It("rejects a body larger than the limit", func() {
			mfRepo := songWith(model.Annotations{Rating: 4})
			big := `{"Rating":8,"Pad":"` + strings.Repeat("x", 70<<10) + `"}`
			Expect(post(big).Code).To(Equal(http.StatusBadRequest))
			Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(4))
		})

		Context("authorization", func() {
			It("refuses a userId naming another user without writing", func() {
				mfRepo := songWith(model.Annotations{Rating: 4})
				w := post(`{"Rating":10,"IsFavorite":true}`, "userId", dto.EncodeID(testID("u2")))
				Expect(w.Code).To(Equal(http.StatusForbidden))
				Expect(mfRepo.Data[testID("s1")].Rating).To(Equal(4))
				Expect(mfRepo.Data[testID("s1")].Starred).To(BeFalse())
			})

			It("accepts the caller's own userId", func() {
				songWith(model.Annotations{})
				Expect(post(`{"Rating":10}`, "userId", dto.EncodeID(testID("u1"))).Code).To(Equal(http.StatusOK))
			})

			It("returns 404 and writes nothing for an album in a library the user can't access", func() {
				albumRepo := ds.Album().(*tests.MockAlbumRepo)
				albumRepo.SetData(model.Albums{{ID: testID("a1"), Name: "One", LibraryID: 2}})
				w := httptest.NewRecorder()
				r := httptest.NewRequest("POST", "/UserItems/a1/UserData", strings.NewReader(`{"Rating":10,"IsFavorite":true}`)).WithContext(ctxUser())
				invoke(api.updateUserItemData, w, withChiURLParam(r, "itemId", dto.EncodeID(testID("a1"))))
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(albumRepo.Data[testID("a1")].Rating).To(Equal(0))
				Expect(albumRepo.Data[testID("a1")].Starred).To(BeFalse())
			})

			It("returns 404 for an unknown item", func() {
				w := post(`{"Rating":10}`)
				Expect(w.Code).To(Equal(http.StatusNotFound))
			})
		})
	})
})

func parseUserData(w *httptest.ResponseRecorder) dto.UserItemDataDto {
	var d dto.UserItemDataDto
	Expect(json.Unmarshal(w.Body.Bytes(), &d)).To(Succeed())
	return d
}

type fakeEventBroker struct {
	http.Handler
	Events []events.Event
}

func (f *fakeEventBroker) SendMessage(_ context.Context, event events.Event) {
	f.Events = append(f.Events, event)
}

func (f *fakeEventBroker) SendBroadcastMessage(_ context.Context, event events.Event) {
	f.Events = append(f.Events, event)
}

var _ events.Broker = (*fakeEventBroker)(nil)
