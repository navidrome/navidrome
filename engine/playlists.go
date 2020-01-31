package engine

import (
	"context"

	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/utils"
)

type Playlists interface {
	GetAll(ctx context.Context) (model.Playlists, error)
	Get(ctx context.Context, id string) (*PlaylistInfo, error)
	Create(ctx context.Context, playlistId, name string, ids []string) error
	Delete(ctx context.Context, playlistId string) error
	Update(ctx context.Context, playlistId string, name *string, idsToAdd []string, idxToRemove []int) error
}

func NewPlaylists(ds model.DataStore) Playlists {
	return &playlists{ds}
}

type playlists struct {
	ds model.DataStore
}

func (p *playlists) Create(ctx context.Context, playlistId, name string, ids []string) error {
	owner := p.getUser(ctx)
	var pls *model.Playlist
	var err error
	// If playlistID is present, override tracks
	if playlistId != "" {
		pls, err = p.ds.Playlist(ctx).Get(playlistId)
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

	return p.ds.Playlist(ctx).Put(pls)
}

func (p *playlists) getUser(ctx context.Context) string {
	user, ok := ctx.Value("user").(*model.User)
	if ok {
		return user.UserName
	}
	return ""
}

func (p *playlists) Delete(ctx context.Context, playlistId string) error {
	pls, err := p.ds.Playlist(ctx).Get(playlistId)
	if err != nil {
		return err
	}

	owner := p.getUser(ctx)
	if owner != pls.Owner {
		return model.ErrNotAuthorized
	}
	return p.ds.Playlist(nil).Delete(playlistId)
}

func (p *playlists) Update(ctx context.Context, playlistId string, name *string, idsToAdd []string, idxToRemove []int) error {
	pls, err := p.ds.Playlist(ctx).Get(playlistId)

	owner := p.getUser(ctx)
	if owner != pls.Owner {
		return model.ErrNotAuthorized
	}

	if err != nil {
		return err
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

	return p.ds.Playlist(ctx).Put(pls)
}

func (p *playlists) GetAll(ctx context.Context) (model.Playlists, error) {
	return p.ds.Playlist(ctx).GetAll(model.QueryOptions{})
}

type PlaylistInfo struct {
	Id        string
	Name      string
	Entries   Entries
	SongCount int
	Duration  int
	Public    bool
	Owner     string
	Comment   string
}

func (p *playlists) Get(ctx context.Context, id string) (*PlaylistInfo, error) {
	pl, err := p.ds.Playlist(ctx).GetWithTracks(id)
	if err != nil {
		return nil, err
	}

	// TODO Use model.Playlist when got rid of Entries
	plsInfo := &PlaylistInfo{
		Id:        pl.ID,
		Name:      pl.Name,
		SongCount: len(pl.Tracks),
		Duration:  pl.Duration,
		Public:    pl.Public,
		Owner:     pl.Owner,
		Comment:   pl.Comment,
	}

	plsInfo.Entries = FromMediaFiles(pl.Tracks)
	return plsInfo, nil
}
