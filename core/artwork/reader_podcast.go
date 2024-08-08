package artwork

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/navidrome/navidrome/model"
)

type podcastReader struct {
	cacheKey
	a       *artwork
	podcast model.Podcast
}

func newPodcastArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID) (*podcastReader, error) {
	pd, err := artwork.ds.Podcast(ctx).Get(artID.ID, false)
	if err != nil {
		return nil, err
	}

	a := &podcastReader{
		a:       artwork,
		podcast: *pd,
	}

	a.cacheKey.artID = artID
	return a, nil
}

func (p *podcastReader) LastUpdated() time.Time {
	return p.lastUpdate
}

func (p *podcastReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	if p.podcast.ImageUrl == "" {
		return nil, "", nil
	}

	imageUrl, err := url.Parse(p.podcast.ImageUrl)
	if err != nil {
		return nil, "", err
	}

	return fromURL(ctx, imageUrl)
}

var _ artworkReader = (*podcastReader)(nil)
