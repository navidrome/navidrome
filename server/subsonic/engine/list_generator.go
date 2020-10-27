package engine

import (
	"context"
	"time"

	"github.com/deluan/navidrome/model"
)

type ListGenerator interface {
	GetNowPlaying(ctx context.Context) (Entries, error)
}

func NewListGenerator(ds model.DataStore, npRepo NowPlayingRepository) ListGenerator {
	return &listGenerator{ds, npRepo}
}

type listGenerator struct {
	ds     model.DataStore
	npRepo NowPlayingRepository
}

func (g *listGenerator) GetNowPlaying(ctx context.Context) (Entries, error) {
	npInfo, err := g.npRepo.GetAll()
	if err != nil {
		return nil, err
	}
	entries := make(Entries, len(npInfo))
	for i, np := range npInfo {
		mf, err := g.ds.MediaFile(ctx).Get(np.TrackID)
		if err != nil {
			return nil, err
		}
		entries[i] = FromMediaFile(mf)
		entries[i].UserName = np.Username
		entries[i].MinutesAgo = int(time.Since(np.Start).Minutes())
		entries[i].PlayerId = np.PlayerId
		entries[i].PlayerName = np.PlayerName
	}
	return entries, nil
}
