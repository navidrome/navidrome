package subsonic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

func (api *Router) GetPlaylists(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	allPls, err := api.ds.Playlist(ctx).GetAll(model.QueryOptions{Sort: "name"})
	if err != nil {
		log.Error(r, err)
		return nil, err
	}
	response := newResponse()
	response.Playlists = &responses.Playlists{
		Playlist: slice.Map(allPls, api.buildPlaylist),
	}
	return response, nil
}

func (api *Router) GetPlaylist(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	return api.getPlaylist(ctx, id)
}

func (api *Router) getPlaylist(ctx context.Context, id string) (*responses.Subsonic, error) {
	pls, err := api.ds.Playlist(ctx).GetWithTracks(id, true)
	if errors.Is(err, model.ErrNotFound) {
		log.Error(ctx, err.Error(), "id", id)
		return nil, newError(responses.ErrorDataNotFound, "playlist not found")
	}
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	response := newResponse()
	response.Playlist = &responses.PlaylistWithSongs{
		Playlist: api.buildPlaylist(*pls),
	}
	response.Playlist.Entry = slice.MapWithArg(pls.MediaFiles(), ctx, childFromMediaFile)
	return response, nil
}

func (api *Router) create(ctx context.Context, playlistId, name string, ids []string) (string, error) {
	err := api.ds.WithTx(func(tx model.DataStore) error {
		owner := getUser(ctx)
		var pls *model.Playlist
		var err error

		if playlistId != "" {
			pls, err = tx.Playlist(ctx).Get(playlistId)
			if err != nil {
				return err
			}
			if owner.ID != pls.OwnerID {
				return model.ErrNotAuthorized
			}
		} else {
			pls = &model.Playlist{Name: name}
			pls.OwnerID = owner.ID
		}
		pls.Tracks = nil
		pls.AddTracks(ids)

		err = tx.Playlist(ctx).Put(pls)
		playlistId = pls.ID
		return err
	})
	return playlistId, err
}

func (api *Router) CreatePlaylist(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	p := req.Params(r)
	songIds, _ := p.Strings("songId")
	playlistId, _ := p.String("playlistId")
	name, _ := p.String("name")
	if playlistId == "" && name == "" {
		return nil, errors.New("required parameter name is missing")
	}
	id, err := api.create(ctx, playlistId, name, songIds)
	if err != nil {
		log.Error(r, err)
		return nil, err
	}
	return api.getPlaylist(ctx, id)
}

func (api *Router) DeletePlaylist(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	err = api.ds.Playlist(r.Context()).Delete(id)
	if errors.Is(err, model.ErrNotAuthorized) {
		return nil, newError(responses.ErrorAuthorizationFail)
	}
	if err != nil {
		log.Error(r, err)
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) UpdatePlaylist(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	playlistId, err := p.String("playlistId")
	if err != nil {
		return nil, err
	}
	songsToAdd, _ := p.Strings("songIdToAdd")
	songIndexesToRemove, _ := p.Ints("songIndexToRemove")
	var plsName *string
	if s, err := p.String("name"); err == nil {
		plsName = &s
	}
	var comment *string
	if s, err := p.String("comment"); err == nil {
		comment = &s
	}
	var public *bool
	if p, err := p.Bool("public"); err == nil {
		public = &p
	}

	log.Debug(r, "Updating playlist", "id", playlistId)
	if plsName != nil {
		log.Trace(r, fmt.Sprintf("-- New Name: '%s'", *plsName))
	}
	log.Trace(r, fmt.Sprintf("-- Adding: '%v'", songsToAdd))
	log.Trace(r, fmt.Sprintf("-- Removing: '%v'", songIndexesToRemove))

	err = api.playlists.Update(r.Context(), playlistId, plsName, comment, public, songsToAdd, songIndexesToRemove)
	if errors.Is(err, model.ErrNotAuthorized) {
		return nil, newError(responses.ErrorAuthorizationFail)
	}
	if err != nil {
		log.Error(r, "Error updating playlist", "id", playlistId, err)
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) buildPlaylist(p model.Playlist) responses.Playlist {
	pls := responses.Playlist{}
	pls.Id = p.ID
	pls.Name = p.Name
	pls.Comment = p.Comment
	pls.SongCount = int32(p.SongCount)
	pls.Owner = p.OwnerName
	pls.Duration = int32(p.Duration)
	pls.Public = p.Public
	pls.Created = p.CreatedAt
	pls.CoverArt = p.CoverArtID().String()
	if p.IsSmartPlaylist() {
		pls.Changed = time.Now()
	} else {
		pls.Changed = p.UpdatedAt
	}
	return pls
}
