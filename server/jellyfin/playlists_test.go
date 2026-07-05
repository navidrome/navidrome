package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
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

	addPlaylistID string
	addIds        []string
	addErr        error

	removePlaylistID string
	removeIds        []string
	removeErr        error
}

func (f *fakePlaylists) Create(_ context.Context, _ string, name string, ids []string) (string, error) {
	f.createdName = name
	f.createdIds = ids
	if f.createErr != nil {
		return "", f.createErr
	}
	return "pl-new", nil
}

func (f *fakePlaylists) GetWithTracks(_ context.Context, _ string) (*model.Playlist, error) {
	if f.getErr != nil {
		return nil, f.getErr
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

var _ = Describe("Playlists", func() {
	var api *Router
	var fp *fakePlaylists

	BeforeEach(func() {
		fp = &fakePlaylists{}
		api = &Router{playlists: fp}
	})

	Describe("createPlaylist", func() {
		It("creates a playlist and returns its id", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists", strings.NewReader(`{"Name":"Mix","Ids":["s1","s2"]}`)).
				WithContext(context.Background())
			api.createPlaylist(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res map[string]string
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res["Id"]).To(Equal("pl-new"))
			Expect(fp.createdName).To(Equal("Mix"))
			Expect(fp.createdIds).To(Equal([]string{"s1", "s2"}))
		})

		It("returns 400 on an invalid JSON body", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists", strings.NewReader(`not json`)).
				WithContext(context.Background())
			api.createPlaylist(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns 500 when the service fails", func() {
			fp.createErr = errors.New("boom")
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists", strings.NewReader(`{"Name":"Mix"}`)).
				WithContext(context.Background())
			api.createPlaylist(w, r)
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
			Expect(res.Items[0].Id).To(Equal("s1"))
			Expect(res.Items[0].Type).To(Equal("Audio"))
			Expect(res.Items[0].PlaylistItemId).To(Equal("1"))
			Expect(res.Items[1].Id).To(Equal("s2"))
			Expect(res.Items[1].PlaylistItemId).To(Equal("2"))
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

	Describe("addToPlaylist", func() {
		It("adds tracks by song id and returns 204", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists/pl1/Items?Ids=s1,s2", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			api.addToPlaylist(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(fp.addPlaylistID).To(Equal("pl1"))
			Expect(fp.addIds).To(Equal([]string{"s1", "s2"}))
		})

		It("returns 404 when the service rejects the request (not found/not owned)", func() {
			fp.addErr = model.ErrNotAuthorized
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists/pl1/Items?Ids=s1", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			api.addToPlaylist(w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("passes no ids (not a spurious empty string) when the Ids param is absent", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Playlists/pl1/Items", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			api.addToPlaylist(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(fp.addPlaylistID).To(Equal("pl1"))
			Expect(fp.addIds).To(BeEmpty())
		})
	})

	Describe("removeFromPlaylist", func() {
		It("removes entries by EntryIds (playlist-track position ids, not song ids) and returns 204", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", "/Playlists/pl1/Items?EntryIds=1,2", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			api.removeFromPlaylist(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(fp.removePlaylistID).To(Equal("pl1"))
			Expect(fp.removeIds).To(Equal([]string{"1", "2"}))
		})

		It("returns 404 when the service rejects the request (not found/not owned)", func() {
			fp.removeErr = model.ErrNotFound
			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", "/Playlists/pl1/Items?EntryIds=1", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			api.removeFromPlaylist(w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("passes no ids (not a spurious empty string) when the EntryIds param is absent", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", "/Playlists/pl1/Items", nil).WithContext(context.Background())
			r = withChiURLParam(r, "playlistId", "pl1")
			api.removeFromPlaylist(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(fp.removePlaylistID).To(Equal("pl1"))
			Expect(fp.removeIds).To(BeEmpty())
		})
	})
})
