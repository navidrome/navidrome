package subsonic

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type PlaylistsController struct {
	ds model.DataStore
}

func NewPlaylistsController(ds model.DataStore) *PlaylistsController {
	return &PlaylistsController{ds: ds}
}

func (c *PlaylistsController) GetPlaylists(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	allPls, err := c.ds.Playlist(ctx).GetAll()
	if err != nil {
		log.Error(r, err)
		return nil, err
	}
	playlists := make([]responses.Playlist, len(allPls))
	for i, p := range allPls {
		playlists[i] = *c.buildPlaylist(p)
	}
	response := newResponse()
	response.Playlists = &responses.Playlists{Playlist: playlists}
	return response, nil
}

func (c *PlaylistsController) GetPlaylist(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	id, err := requiredParamString(r, "id")
	if err != nil {
		return nil, err
	}
	pls, err := c.ds.Playlist(ctx).Get(id)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, err.Error(), "id", id)
		return nil, newError(responses.ErrorDataNotFound, "Directory not found")
	case err != nil:
		log.Error(r, err)
		return nil, err
	}

	response := newResponse()
	response.Playlist = c.buildPlaylistWithSongs(ctx, pls)
	return response, nil
}

func (c *PlaylistsController) create(ctx context.Context, playlistId, name string, ids []string) error {
	return c.ds.WithTx(func(tx model.DataStore) error {
		owner := getUser(ctx)
		var pls *model.Playlist
		var err error

		// If playlistID is present, override tracks
		if playlistId != "" {
			pls, err = tx.Playlist(ctx).Get(playlistId)
			if err != nil {
				return err
			}
			if owner != pls.Owner {
				return model.ErrNotAuthorized
			}
			pls.Tracks = nil
		} else {
			pls = &model.Playlist{
				Name:  name,
				Owner: owner,
			}
		}
		for _, id := range ids {
			pls.Tracks = append(pls.Tracks, model.MediaFile{ID: id})
		}

		return tx.Playlist(ctx).Put(pls)
	})
}

func (c *PlaylistsController) CreatePlaylist(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	songIds := utils.ParamStrings(r, "songId")
	playlistId := utils.ParamString(r, "playlistId")
	name := utils.ParamString(r, "name")
	if playlistId == "" && name == "" {
		return nil, errors.New("required parameter name is missing")
	}
	err := c.create(r.Context(), playlistId, name, songIds)
	if err != nil {
		log.Error(r, err)
		return nil, err
	}
	return newResponse(), nil
}

func (c *PlaylistsController) delete(ctx context.Context, playlistId string) error {
	return c.ds.WithTx(func(tx model.DataStore) error {
		pls, err := tx.Playlist(ctx).Get(playlistId)
		if err != nil {
			return err
		}

		owner := getUser(ctx)
		if owner != pls.Owner {
			return model.ErrNotAuthorized
		}
		return tx.Playlist(ctx).Delete(playlistId)
	})
}

func (c *PlaylistsController) DeletePlaylist(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := requiredParamString(r, "id")
	if err != nil {
		return nil, err
	}
	err = c.delete(r.Context(), id)
	if err == model.ErrNotAuthorized {
		return nil, newError(responses.ErrorAuthorizationFail)
	}
	if err != nil {
		log.Error(r, err)
		return nil, err
	}
	return newResponse(), nil
}

func (p *PlaylistsController) update(ctx context.Context, playlistId string, name *string, idsToAdd []string, idxToRemove []int) error {
	return p.ds.WithTx(func(tx model.DataStore) error {
		pls, err := tx.Playlist(ctx).Get(playlistId)
		if err != nil {
			return err
		}

		owner := getUser(ctx)
		if owner != pls.Owner {
			return model.ErrNotAuthorized
		}

		if name != nil {
			pls.Name = *name
		}
		newTracks := model.MediaFiles{}
		for i, t := range pls.Tracks {
			if utils.IntInSlice(i, idxToRemove) {
				continue
			}
			newTracks = append(newTracks, t)
		}

		for _, id := range idsToAdd {
			newTracks = append(newTracks, model.MediaFile{ID: id})
		}
		pls.Tracks = newTracks

		return tx.Playlist(ctx).Put(pls)
	})
}

func (c *PlaylistsController) UpdatePlaylist(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	playlistId, err := requiredParamString(r, "playlistId")
	if err != nil {
		return nil, err
	}
	songsToAdd := utils.ParamStrings(r, "songIdToAdd")
	songIndexesToRemove := utils.ParamInts(r, "songIndexToRemove")

	var plsName *string
	if len(r.URL.Query()["name"]) > 0 {
		s := r.URL.Query()["name"][0]
		plsName = &s
	}

	log.Debug(r, "Updating playlist", "id", playlistId)
	if plsName != nil {
		log.Trace(r, fmt.Sprintf("-- New Name: '%s'", *plsName))
	}
	log.Trace(r, fmt.Sprintf("-- Adding: '%v'", songsToAdd))
	log.Trace(r, fmt.Sprintf("-- Removing: '%v'", songIndexesToRemove))

	err = c.update(r.Context(), playlistId, plsName, songsToAdd, songIndexesToRemove)
	if err == model.ErrNotAuthorized {
		return nil, newError(responses.ErrorAuthorizationFail)
	}
	if err != nil {
		log.Error(r, err)
		return nil, err
	}
	return newResponse(), nil
}

func (c *PlaylistsController) buildPlaylistWithSongs(ctx context.Context, p *model.Playlist) *responses.PlaylistWithSongs {
	pls := &responses.PlaylistWithSongs{
		Playlist: *c.buildPlaylist(*p),
	}
	pls.Entry = childrenFromMediaFiles(ctx, p.Tracks)
	return pls
}

func (c *PlaylistsController) buildPlaylist(p model.Playlist) *responses.Playlist {
	pls := &responses.Playlist{}
	pls.Id = p.ID
	pls.Name = p.Name
	pls.Comment = p.Comment
	pls.SongCount = p.SongCount
	pls.Owner = p.Owner
	pls.Duration = int(p.Duration)
	pls.Public = p.Public
	pls.Created = p.CreatedAt
	pls.Changed = p.UpdatedAt
	return pls
}
