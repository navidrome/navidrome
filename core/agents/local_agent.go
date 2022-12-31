package agents

import (
	"context"

	"github.com/navidrome/navidrome/model"
)

const LocalAgentName = "local"

const (
	localBiography = "Biography not available"
)

type localAgent struct{}

func localsConstructor(_ model.DataStore) Interface {
	return &localAgent{}
}

func (p *localAgent) AgentName() string {
	return LocalAgentName
}

func (p *localAgent) GetBiography(ctx context.Context, id, name, mbid string) (string, error) {
	return localBiography, nil
}

func (p *localAgent) GetTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]Song, error) {
	return nil, nil // TODO return 5-stars and liked songs sorted by playCount
}

func init() {
	Register(LocalAgentName, localsConstructor)
}
