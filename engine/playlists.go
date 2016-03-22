package engine

import (
	"github.com/deluan/gosonic/domain"
)

type Playlists interface {
	GetAll() (domain.Playlists, error)
	Get(id string) (*PlaylistInfo, error)
}

func NewPlaylists(pr domain.PlaylistRepository, mr domain.MediaFileRepository) Playlists {
	return &playlists{pr, mr}
}

type playlists struct {
	plsRepo   domain.PlaylistRepository
	mfileRepo domain.MediaFileRepository
}

func (p *playlists) GetAll() (domain.Playlists, error) {
	return p.plsRepo.GetAll(domain.QueryOptions{})
}

type PlaylistInfo struct {
	Id        string
	Name      string
	Entries   Entries
	SongCount int
	Duration  int
	Public    bool
	Owner     string
}

func (p *playlists) Get(id string) (*PlaylistInfo, error) {
	pl, err := p.plsRepo.Get(id)
	if err == domain.ErrNotFound {
		return nil, ErrDataNotFound
	}
	if err != nil {
		return nil, err
	}

	pinfo := &PlaylistInfo{
		Id:        pl.Id,
		Name:      pl.Name,
		SongCount: len(pl.Tracks),
		Duration:  pl.Duration,
		Public:    pl.Public,
		Owner:     pl.Owner,
	}
	pinfo.Entries = make(Entries, len(pl.Tracks))

	// TODO Optimize: Get all tracks at once
	for i, mfId := range pl.Tracks {
		mf, err := p.mfileRepo.Get(mfId)
		if err != nil {
			return nil, err
		}
		pinfo.Entries[i] = FromMediaFile(mf)
		pinfo.Entries[i].Track = 0
	}

	return pinfo, nil
}
