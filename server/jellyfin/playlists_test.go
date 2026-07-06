package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/filter"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakePlaylists is a local fake for core/playlists.Playlists. It embeds the interface so
// unimplemented methods aren't needed here; only the ones this test exercises are overridden.
type fakePlaylists struct {
	playlists.Playlists

	createdName string
	createdIds  []string
	createErr   error

	getPls *model.Playlist
	getErr error

	getByIDPls *model.Playlist
	getByIDErr error

	addPlaylistID string
	addIds        []string
	addErr        error

	removePlaylistID string
	removeIds        []string
	removeErr        error

	setImagePlaylistID string
	setImageBytes      []byte
	setImageExt        string
	setImageErr        error

	removeImagePlaylistID string
	removeImageErr        error

	deletePlaylistID string
	deleteErr        error
}

func (f *fakePlaylists) Delete(_ context.Context, id string) error {
	f.deletePlaylistID = id
	return f.deleteErr
}

func (f *fakePlaylists) Create(_ context.Context, _ string, name string, ids []string) (string, error) {
	f.createdName = name
	f.createdIds = ids
	if f.createErr != nil {
		return "", f.createErr
	}
	return "pl-new", nil
}

// Get defaults to model.ErrNotFound when getByIDPls/getByIDErr aren't set, matching the real
// service's behavior for a missing or inaccessible playlist and letting getItem tests that don't
// care about playlists leave it unconfigured.
func (f *fakePlaylists) Get(_ context.Context, _ string) (*model.Playlist, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	if f.getByIDPls == nil {
		return nil, model.ErrNotFound
	}
	return f.getByIDPls, nil
}

func (f *fakePlaylists) GetWithTracks(_ context.Context, _ string) (*model.Playlist, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getPls == nil {
		return nil, model.ErrNotFound // mirror the real repo: never (nil, nil)
	}
	return f.getPls, nil
}

func (f *fakePlaylists) AddTracks(_ context.Context, playlistID string, ids []string) (int, error) {
	f.addPlaylistID = playlistID
	f.addIds = ids
	return len(ids), f.addErr
}

func (f *fakePlaylists) RemoveTracks(_ context.Context, playlistID string, trackIds []string) error {
	f.removePlaylistID = playlistID
	f.removeIds = trackIds
	return f.removeErr
}

func (f *fakePlaylists) SetImage(_ context.Context, playlistID string, reader io.Reader, ext string) error {
	f.setImagePlaylistID = playlistID
	f.setImageExt = ext
	if reader != nil {
		f.setImageBytes, _ = io.ReadAll(reader)
	}
	return f.setImageErr
}

func (f *fakePlaylists) RemoveImage(_ context.Context, playlistID string) error {
	f.removeImagePlaylistID = playlistID
	return f.removeImageErr
}

var _ = Describe("Playlists", func() {
	var api *Router
	var fp *fakePlaylists

	BeforeEach(func() {
		fp = &fakePlaylists{}
		api = &Router{ds: &tests.MockDataStore{}, playlists: fp}
	})

	Describe("createPlaylist", func() {
		It("creates a playlist and returns its id", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists", strings.NewReader(`{"Name":"Mix","Ids":["s1","s2"]}`)).
				WithContext(context.Background())
			invoke(api.createPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res map[string]string
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res["Id"]).To(Equal(dto.EncodeID("pl-new")))
			Expect(fp.createdName).To(Equal("Mix"))
			Expect(fp.createdIds).To(Equal([]string{"s1", "s2"}))
		})

		It("returns 400 on an invalid JSON body", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists", strings.NewReader(`not json`)).
				WithContext(context.Background())
			invoke(api.createPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns 500 when the service fails", func() {
			fp.createErr = errors.New("boom")
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists", strings.NewReader(`{"Name":"Mix"}`)).
				WithContext(context.Background())
			invoke(api.createPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("getPlaylistItems", func() {
		It("maps playlist tracks to Audio BaseItemDtos, tagging each with its PlaylistItemId", func() {
			fp.getPls = &model.Playlist{
				ID: "pl1",
				Tracks: model.PlaylistTracks{
					{ID: "1", MediaFileID: "s1", PlaylistID: "pl1", MediaFile: model.MediaFile{ID: "s1", Title: "Song One"}},
					{ID: "2", MediaFileID: "s2", PlaylistID: "pl1", MediaFile: model.MediaFile{ID: "s2", Title: "Song Two"}},
				},
			}
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Playlists/pl1/Items", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			api.getPlaylistItems(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.TotalRecordCount).To(Equal(2))
			Expect(res.Items).To(HaveLen(2))
			Expect(res.Items[0].Id).To(Equal(dto.EncodeID("s1")))
			Expect(res.Items[0].Type).To(Equal("Audio"))
			Expect(res.Items[0].PlaylistItemId).To(Equal(dto.EncodeID("1")))
			Expect(res.Items[1].Id).To(Equal(dto.EncodeID("s2")))
			Expect(res.Items[1].PlaylistItemId).To(Equal(dto.EncodeID("2")))
		})

		It("returns 404 for a non-owned or absent playlist", func() {
			fp.getErr = model.ErrNotFound
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Playlists/missing/Items", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "missing")
			api.getPlaylistItems(w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("container id expansion", func() {
		var ds *tests.MockDataStore
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
			ds = &tests.MockDataStore{}
			api = &Router{ds: ds, playlists: fp}
		})

		createWith := func(id string) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists", strings.NewReader(`{"Name":"Mix","Ids":["`+id+`"]}`)).
				WithContext(ctx)
			invoke(api.createPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
		}

		It("passes a bare song id through unchanged", func() {
			ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{{ID: "s1"}})
			createWith("s1")
			Expect(fp.createdIds).To(Equal([]string{"s1"}))
		})

		It("expands an album id into its songs, filtered by album", func() {
			ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "al1"}})
			ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", AlbumID: "al1"}, {ID: "s2", AlbumID: "al1"},
			})
			createWith("al1")
			Expect(fp.createdIds).To(Equal([]string{"s1", "s2"}))
			Expect(ds.MediaFile(ctx).(*tests.MockMediaFileRepo).Options.Filters).To(Equal(filter.SongsByAlbum("al1").Filters))
		})

		It("expands an artist id into its songs", func() {
			ds.Artist(ctx).(*tests.MockArtistRepo).SetData(model.Artists{{ID: "ar1"}})
			ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{{ID: "s1"}, {ID: "s2"}})
			createWith("ar1")
			Expect(fp.createdIds).To(Equal([]string{"s1", "s2"}))
			Expect(ds.MediaFile(ctx).(*tests.MockMediaFileRepo).Options.Filters).To(Equal(filter.SongsByArtistID("ar1").Filters))
		})

		It("expands a playlist id into its tracks' media file ids", func() {
			fp.getPls = &model.Playlist{ID: "pl9", Tracks: model.PlaylistTracks{
				{ID: "1", MediaFileID: "s3"}, {ID: "2", MediaFileID: "s4"},
			}}
			createWith("pl9")
			Expect(fp.createdIds).To(Equal([]string{"s3", "s4"}))
		})
	})

	Describe("getPlaylist", func() {
		It("returns OpenAccess from Public and item ids (encoded media file ids, not entry ids)", func() {
			fp.getPls = &model.Playlist{
				ID:     "pl1",
				Public: true,
				Tracks: model.PlaylistTracks{
					{ID: "1", MediaFileID: "s1", PlaylistID: "pl1", MediaFile: model.MediaFile{ID: "s1"}},
					{ID: "2", MediaFileID: "s2", PlaylistID: "pl1", MediaFile: model.MediaFile{ID: "s2"}},
				},
			}
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Playlists/pl1", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			invoke(api.getPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.PlaylistInfo
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.OpenAccess).To(BeTrue())
			Expect(res.Shares).To(BeEmpty())
			Expect(res.ItemIds).To(Equal([]string{dto.EncodeID("s1"), dto.EncodeID("s2")}))
		})

		It("returns 404 for a non-owned or absent playlist", func() {
			fp.getErr = model.ErrNotFound
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Playlists/missing", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "missing")
			invoke(api.getPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("deleteItem", func() {
		deleteReq := func(id string) *http.Request {
			r := httptest.NewRequest("DELETE", "/Items/"+dto.EncodeID(id), nil).WithContext(context.Background())
			return withChiURLParam(r, "itemId", dto.EncodeID(id))
		}

		It("deletes the playlist and returns 204", func() {
			w := httptest.NewRecorder()
			api.deleteItem(w, deleteReq("pl1"))
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(fp.deletePlaylistID).To(Equal("pl1"))
		})

		It("returns 403 when the user doesn't own the playlist", func() {
			fp.deleteErr = model.ErrNotAuthorized
			w := httptest.NewRecorder()
			api.deleteItem(w, deleteReq("pl1"))
			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("returns 404 for a missing playlist or non-playlist id", func() {
			fp.deleteErr = model.ErrNotFound
			w := httptest.NewRecorder()
			api.deleteItem(w, deleteReq("al1"))
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 500 on an unexpected error", func() {
			fp.deleteErr = errors.New("boom")
			w := httptest.NewRecorder()
			api.deleteItem(w, deleteReq("pl1"))
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("addToPlaylist", func() {
		It("adds tracks by song id from the lowercase ids param real Jellyfin clients send", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists/pl1/Items?ids=s1,s2", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			invoke(api.addToPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(fp.addPlaylistID).To(Equal("pl1"))
			Expect(fp.addIds).To(Equal([]string{"s1", "s2"}))
		})

		It("accepts a PascalCase Ids param (case-folded by the middleware)", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists/pl1/Items?Ids=s1,s2", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			invoke(api.addToPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(fp.addIds).To(Equal([]string{"s1", "s2"}))
		})

		It("returns 404 when the service rejects the request (not found/not owned)", func() {
			fp.addErr = model.ErrNotAuthorized
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists/pl1/Items?ids=s1", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			invoke(api.addToPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("passes no ids (not a spurious empty string) when the ids param is absent", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists/pl1/Items", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			invoke(api.addToPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(fp.addPlaylistID).To(Equal("pl1"))
			Expect(fp.addIds).To(BeEmpty())
		})
	})

	Describe("removeFromPlaylist", func() {
		It("removes entries by the lowercase entryIds param real Jellyfin clients send (playlist-track position ids, not song ids)", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", "/Playlists/pl1/Items?entryIds=1,2", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			invoke(api.removeFromPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(fp.removePlaylistID).To(Equal("pl1"))
			Expect(fp.removeIds).To(Equal([]string{"1", "2"}))
		})

		It("accepts a PascalCase EntryIds param (case-folded by the middleware)", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", "/Playlists/pl1/Items?EntryIds=1,2", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			invoke(api.removeFromPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(fp.removeIds).To(Equal([]string{"1", "2"}))
		})

		It("returns 404 when the service rejects the request (not found/not owned)", func() {
			fp.removeErr = model.ErrNotFound
			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", "/Playlists/pl1/Items?entryIds=1", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			invoke(api.removeFromPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("passes no ids (not a spurious empty string) when the entryIds param is absent", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", "/Playlists/pl1/Items", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			invoke(api.removeFromPlaylist, w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(fp.removePlaylistID).To(Equal("pl1"))
			Expect(fp.removeIds).To(BeEmpty())
		})
	})

	Describe("getPlaylistUsers", func() {
		It("returns the current user with CanEdit true", func() {
			w := httptest.NewRecorder()
			ctx := request.WithUser(context.Background(), model.User{ID: "u1", UserName: "alice"})
			r := httptest.NewRequest("GET", "/Playlists/pl1/Users", nil).WithContext(ctx)
			r = withChiURLParam(r, "playlistId", "pl1")
			api.getPlaylistUsers(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res []dto.PlaylistUserPermissions
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res).To(Equal([]dto.PlaylistUserPermissions{{UserId: "u1", CanEdit: true}}))
		})
	})

	Describe("getPlaylistUser", func() {
		It("returns CanEdit true for the requested user", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Playlists/pl1/Users/u1", nil).WithContext(context.Background())
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("playlistId", "pl1")
			rctx.URLParams.Add("userId", "u1")
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
			api.getPlaylistUser(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.PlaylistUserPermissions
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res).To(Equal(dto.PlaylistUserPermissions{UserId: "u1", CanEdit: true}))
		})
	})
})
