package agents

import (
	"context"
	"path/filepath"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
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

func (p *localAgent) GetImages(_ context.Context, id, name, mbid string) ([]ArtistImage, error) {
	return []ArtistImage{
		p.artistImage(id, 300),
		p.artistImage(id, 174),
		p.artistImage(id, 64),
	}, nil
}

func (p *localAgent) artistImage(id string, size int) ArtistImage {
	return ArtistImage{
		filepath.Join(consts.URLPathPublicImages, artwork.Public(model.NewArtworkID(model.KindArtistArtwork, id), size)),
		size,
	}
}

func init() {
	Register(LocalAgentName, localsConstructor)
}
