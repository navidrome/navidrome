package engine

import (
	"context"
	"sort"

	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/log"
)

type Playlists interface {
	GetAll() (domain.Playlists, error)
	Get(id string) (*PlaylistInfo, error)
	Create(ctx context.Context, name string, ids []string) error
	Delete(ctx context.Context, playlistId string) error
	Update(playlistId string, name *string, idsToAdd []string, idxToRemove []int) error
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
	Comment   string
}

func (p *playlists) Create(ctx context.Context, name string, ids []string) error {
	pid, err := p.itunes.CreatePlaylist(name, ids)
	if err != nil {
		return err
	}
	log.Info(ctx, "Created playlist", "playlist", name, "id", pid)
	return nil
}

func (p *playlists) Delete(ctx context.Context, playlistId string) error {
	err := p.itunes.DeletePlaylist(playlistId)
	if err != nil {
		return err
	}
	log.Info(ctx, "Deleted playlist", "id", playlistId)
	return nil
}

func (p *playlists) Update(playlistId string, name *string, idsToAdd []string, idxToRemove []int) error {
	pl, err := p.plsRepo.Get(playlistId)
	if err != nil {
		return err
	}
	if name != nil {
		pl.Name = *name
		err := p.itunes.RenamePlaylist(pl.Id, pl.Name)
		if err != nil {
			return err
		}
	}
	if len(idsToAdd) > 0 || len(idxToRemove) > 0 {
		sort.Sort(sort.Reverse(sort.IntSlice(idxToRemove)))
		for _, i := range idxToRemove {
			pl.Tracks, pl.Tracks[len(pl.Tracks)-1] = append(pl.Tracks[:i], pl.Tracks[i+1:]...), ""
		}
		pl.Tracks = append(pl.Tracks, idsToAdd...)
		err := p.itunes.UpdatePlaylist(pl.Id, pl.Tracks)
		if err != nil {
			return err
		}
	}
	p.plsRepo.Put(pl) // Ignores errors, as any changes will be overridden in the next scan
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
		Comment:   pl.Comment,
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
