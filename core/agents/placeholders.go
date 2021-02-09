package agents

import (
	"context"
)

const PlaceholderAgentName = "placeholder"

const (
	placeholderArtistImageSmallUrl  = "https://lastfm.freetls.fastly.net/i/u/64s/2a96cbd8b46e442fc41c2b86b821562f.png"
	placeholderArtistImageMediumUrl = "https://lastfm.freetls.fastly.net/i/u/174s/2a96cbd8b46e442fc41c2b86b821562f.png"
	placeholderArtistImageLargeUrl  = "https://lastfm.freetls.fastly.net/i/u/300x300/2a96cbd8b46e442fc41c2b86b821562f.png"
	placeholderBiography            = "Biography not available"
)

type placeholderAgent struct{}

func placeholdersConstructor(ctx context.Context) Interface {
	return &placeholderAgent{}
}

func (p *placeholderAgent) AgentName() string {
	return PlaceholderAgentName
}

func (p *placeholderAgent) GetBiography(id, name, mbid string) (string, error) {
	return placeholderBiography, nil
}

func (p *placeholderAgent) GetImages(id, name, mbid string) ([]ArtistImage, error) {
	return []ArtistImage{
		{placeholderArtistImageLargeUrl, 300},
		{placeholderArtistImageMediumUrl, 174},
		{placeholderArtistImageSmallUrl, 64},
	}, nil
}

func init() {
	Register(PlaceholderAgentName, placeholdersConstructor)
}
