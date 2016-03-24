package engine

import (
	"fmt"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/itunesbridge"
)

type Playlists interface {
	GetAll() (domain.Playlists, error)
	Get(id string) (*PlaylistInfo, error)
	Create(name string, ids []string) error
}

func NewPlaylists(itunes itunesbridge.ItunesControl, pr domain.PlaylistRepository, mr domain.MediaFileRepository) Playlists {
	return &playlists{itunes, pr, mr}
}

type playlists struct {
	itunes    itunesbridge.ItunesControl
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

func (p *playlists) Create(name string, ids []string) error {
	pid, err := p.itunes.CreatePlaylist(name, ids)
	if err != nil {
		return err
	}
	beego.Info(fmt.Sprintf("Created playlist '%s' with id '%s'", name, pid))
	return nil
}

func (p *playlists) Get(id string) (*PlaylistInfo, error) {
	pl, err := p.plsRepo.Get(id)
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
	}

	return pinfo, nil
}
