package agents

import (
	"context"

	"github.com/navidrome/navidrome/model"
)

const PlaceholderAgentName = "placeholder"

const (
	placeholderArtistImageSmallUrl  = ""
	placeholderArtistImageMediumUrl = ""
	placeholderArtistImageLargeUrl  = ""
	placeholderBiography            = "Biography not available"
)

type placeholderAgent struct{}

func placeholdersConstructor(ds model.DataStore) Interface {
	return &placeholderAgent{}
}

func (p *placeholderAgent) AgentName() string {
	return PlaceholderAgentName
}

func (p *placeholderAgent) GetBiography(ctx context.Context, id, name, mbid string) (string, error) {
	return placeholderBiography, nil
}

func (p *placeholderAgent) GetImages(ctx context.Context, id, name, mbid string) ([]ArtistImage, error) {
	return []ArtistImage{
		{placeholderArtistImageLargeUrl, 300},
		{placeholderArtistImageMediumUrl, 174},
		{placeholderArtistImageSmallUrl, 64},
	}, nil
}

func init() {
	Register(PlaceholderAgentName, placeholdersConstructor)
}
