package api

import (
	"fmt"
	"net/http"

	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/log"
)

type PlaylistsController struct {
	pls engine.Playlists
}

func NewPlaylistsController(pls engine.Playlists) *PlaylistsController {
	return &PlaylistsController{pls: pls}
}

func (c *PlaylistsController) GetPlaylists(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	allPls, err := c.pls.GetAll()
	if err != nil {
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal error")
	}
	playlists := make([]responses.Playlist, len(allPls))
	for i, p := range allPls {
		playlists[i].Id = p.Id
		playlists[i].Name = p.Name
		playlists[i].Comment = p.Comment
		playlists[i].SongCount = len(p.Tracks)
		playlists[i].Duration = p.Duration
		playlists[i].Owner = p.Owner
		playlists[i].Public = p.Public
	}
	response := NewEmpty()
	response.Playlists = &responses.Playlists{Playlist: playlists}
	return response, nil
}

func (c *PlaylistsController) GetPlaylist(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := RequiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}
	pinfo, err := c.pls.Get(id)
	switch {
	case err == domain.ErrNotFound:
		log.Error(r, err.Error(), "id", id)
		return nil, NewError(responses.ErrorDataNotFound, "Directory not found")
	case err != nil:
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewEmpty()
	response.Playlist = c.buildPlaylist(pinfo)
	return response, nil
}

func (c *PlaylistsController) CreatePlaylist(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	songIds, err := RequiredParamStrings(r, "songId", "Required parameter songId is missing")
	if err != nil {
		return nil, err
	}
	name, err := RequiredParamString(r, "name", "Required parameter name is missing")
	if err != nil {
		return nil, err
	}
	err = c.pls.Create(r.Context(), name, songIds)
	if err != nil {
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}
	return NewEmpty(), nil
}

func (c *PlaylistsController) DeletePlaylist(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := RequiredParamString(r, "id", "Required parameter id is missing")
	if err != nil {
		return nil, err
	}
	err = c.pls.Delete(r.Context(), id)
	if err != nil {
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}
	return NewEmpty(), nil
}

func (c *PlaylistsController) UpdatePlaylist(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	playlistId, err := RequiredParamString(r, "playlistId", "Required parameter playlistId is missing")
	if err != nil {
		return nil, err
	}
	songsToAdd := ParamStrings(r, "songIdToAdd")
	songIndexesToRemove := ParamInts(r, "songIndexToRemove")

	var pname *string
	if len(r.URL.Query()["name"]) > 0 {
		s := r.URL.Query()["name"][0]
		pname = &s
	}

	log.Info(r, "Updating playlist", "id", playlistId)
	if pname != nil {
		log.Debug(r, fmt.Sprintf("-- New Name: '%s'", *pname))
	}
	log.Debug(r, fmt.Sprintf("-- Adding: '%v'", songsToAdd))
	log.Debug(r, fmt.Sprintf("-- Removing: '%v'", songIndexesToRemove))

	err = c.pls.Update(playlistId, pname, songsToAdd, songIndexesToRemove)
	if err != nil {
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}
	return NewEmpty(), nil
}

func (c *PlaylistsController) buildPlaylist(d *engine.PlaylistInfo) *responses.PlaylistWithSongs {
	pls := &responses.PlaylistWithSongs{}
	pls.Id = d.Id
	pls.Name = d.Name
	pls.SongCount = d.SongCount
	pls.Owner = d.Owner
	pls.Duration = d.Duration
	pls.Public = d.Public

	pls.Entry = ToChildren(d.Entries)
	return pls
}
