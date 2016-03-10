package engine

import (
	"github.com/deluan/gosonic/domain"
)

type Playlists interface {
	GetAll() (*domain.Playlists, error)
	Get(id string) (*PlaylistInfo, error)
}

func NewPlaylists(pr domain.PlaylistRepository, mr domain.MediaFileRepository) Playlists {
	return playlists{pr, mr}
}

type playlists struct {
	plsRepo   domain.PlaylistRepository
	mfileRepo domain.MediaFileRepository
}

func (p playlists) GetAll() (*domain.Playlists, error) {
	return p.plsRepo.GetAll(domain.QueryOptions{})
}

type PlaylistInfo struct {
	Id      string
	Name    string
	Entries []Child
}

func (p playlists) Get(id string) (*PlaylistInfo, error) {
	pl, err := p.plsRepo.Get(id)
	if err != nil {
		return nil, err
	}

	if pl == nil {
		return nil, ErrDataNotFound
	}

	pinfo := &PlaylistInfo{Id: pl.Id, Name: pl.Name}
	pinfo.Entries = make([]Child, len(pl.Tracks))

	// TODO Optimize: Get all tracks at once
	for i, mfId := range pl.Tracks {
		mf, err := p.mfileRepo.Get(mfId)
		if err != nil {
			return nil, err
		}
		pinfo.Entries[i].Id = mf.Id
		pinfo.Entries[i].Title = mf.Title
		pinfo.Entries[i].IsDir = false
		pinfo.Entries[i].Parent = mf.AlbumId
		pinfo.Entries[i].Album = mf.Album
		pinfo.Entries[i].Year = mf.Year
		pinfo.Entries[i].Artist = mf.Artist
		pinfo.Entries[i].Genre = mf.Genre
		//pinfo.Entries[i].Track = mf.TrackNumber
		pinfo.Entries[i].Duration = mf.Duration
		pinfo.Entries[i].Size = mf.Size
		pinfo.Entries[i].Suffix = mf.Suffix
		pinfo.Entries[i].BitRate = mf.BitRate
		if mf.Starred {
			pinfo.Entries[i].Starred = mf.UpdatedAt
		}
		if mf.HasCoverArt {
			pinfo.Entries[i].CoverArt = mf.Id
		}
		pinfo.Entries[i].ContentType = mf.ContentType()
	}

	return pinfo, nil
}
