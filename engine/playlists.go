package engine

import (
	"context"

	"github.com/cloudsonic/sonic-server/model"
)

type Playlists interface {
	GetAll() (model.Playlists, error)
	Get(id string) (*PlaylistInfo, error)
	Create(ctx context.Context, name string, ids []string) error
	Delete(ctx context.Context, playlistId string) error
	Update(playlistId string, name *string, idsToAdd []string, idxToRemove []int) error
}

func NewPlaylists(ds model.DataStore) Playlists {
	return &playlists{ds}
}

type playlists struct {
	ds model.DataStore
}

func (p *playlists) GetAll() (model.Playlists, error) {
	return p.ds.Playlist().GetAll(model.QueryOptions{})
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

func (p *playlists) Create(ctx context.Context, name string, ids []string) error {
	// TODO
	return nil
}

func (p *playlists) Delete(ctx context.Context, playlistId string) error {
	// TODO
	return nil
}

func (p *playlists) Update(playlistId string, name *string, idsToAdd []string, idxToRemove []int) error {
	// TODO
	return nil
}

func (p *playlists) Get(id string) (*PlaylistInfo, error) {
	pl, err := p.ds.Playlist().Get(id)
	if err != nil {
		return nil, err
	}

	pinfo := &PlaylistInfo{
		Id:        pl.ID,
		Name:      pl.Name,
		SongCount: len(pl.Tracks), // TODO Use model.Playlist
		Duration:  pl.Duration,
		Public:    pl.Public,
		Owner:     pl.Owner,
		Comment:   pl.Comment,
	}
	pinfo.Entries = make(Entries, len(pl.Tracks))

	// TODO Optimize: Get all tracks at once
	for i, mfId := range pl.Tracks {
		mf, err := p.ds.MediaFile().Get(mfId)
		if err != nil {
			return nil, err
		}
		pinfo.Entries[i] = FromMediaFile(mf)
	}

	return pinfo, nil
}
