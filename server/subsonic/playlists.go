package subsonic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

func (api *Router) GetPlaylists(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	allPls, err := api.playlists.GetAll(ctx, model.QueryOptions{Sort: "name"})
	if err != nil {
		log.Error(r, err)
		return nil, err
	}
	response := newResponse()
	response.Playlists = &responses.Playlists{
		Playlist: slice.MapWithArg(allPls, ctx, api.buildPlaylist),
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
	pls, err := api.playlists.GetWithTracks(ctx, id)
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
		Playlist: api.buildPlaylist(ctx, *pls),
	}
	response.Playlist.Entry = slice.MapWithArg(pls.MediaFiles(), ctx, childFromMediaFile)
	return response, nil
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
	id, err := api.playlists.Create(ctx, playlistId, name, songIds)
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
	err = api.playlists.Delete(r.Context(), id)
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
	comment := p.StringPtr("comment")
	public := p.BoolPtr("public")

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

func (api *Router) buildPlaylist(ctx context.Context, p model.Playlist) responses.Playlist {
	pls := responses.Playlist{}
	pls.Id = p.ID
	pls.Name = p.Name
	pls.SongCount = int32(p.SongCount)
	pls.Duration = int32(p.Duration)
	pls.Created = p.CreatedAt
	if p.IsSmartPlaylist() {
		if p.EvaluatedAt != nil {
			pls.Changed = *p.EvaluatedAt
		} else {
			pls.Changed = time.Now()
		}
	} else {
		pls.Changed = p.UpdatedAt
	}

	player, ok := request.PlayerFrom(ctx)
	if ok && isClientInList(conf.Server.Subsonic.MinimalClients, player.Client) {
		return pls
	}

	pls.Comment = p.Comment
	pls.Owner = p.OwnerName
	pls.Public = p.Public
	pls.CoverArt = p.CoverArtID().String()
	pls.OpenSubsonicPlaylist = buildOSPlaylist(ctx, p)

	return pls
}

func buildOSPlaylist(ctx context.Context, p model.Playlist) *responses.OpenSubsonicPlaylist {
	pls := responses.OpenSubsonicPlaylist{}

	if p.IsSmartPlaylist() {
		pls.Readonly = true

		if p.EvaluatedAt != nil {
			pls.ValidUntil = P(p.EvaluatedAt.Add(conf.Server.SmartPlaylistRefreshDelay))
		}
	} else {
		user, ok := request.UserFrom(ctx)
		pls.Readonly = !ok || p.OwnerID != user.ID
	}

	return &pls
}
