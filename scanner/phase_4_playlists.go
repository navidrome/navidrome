package scanner

import (
	"context"

	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
)

type phasePlaylists struct {
	ctx       context.Context
	scanState *scanState
	ds        model.DataStore
	pls       core.Playlists
}

func createPhasePlaylists(ctx context.Context, scanState *scanState, ds model.DataStore, pls core.Playlists) *phasePlaylists {
	return &phasePlaylists{
		ctx:       ctx,
		scanState: scanState,
		ds:        ds,
		pls:       pls,
	}
}

func (p *phasePlaylists) description() string {
	//TODO implement me
	panic("implement me")
}

func (p *phasePlaylists) producer() ppl.Producer[*model.Playlist] {
	//TODO implement me
	panic("implement me")
}

func (p *phasePlaylists) stages() []ppl.Stage[*model.Playlist] {
	//TODO implement me
	panic("implement me")
}

func (p *phasePlaylists) finalize(err error) error {
	//TODO implement me
	panic("implement me")
}

var _ phase[*model.Playlist] = (*phasePlaylists)(nil)
