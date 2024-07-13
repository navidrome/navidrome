package artwork

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/navidrome/navidrome/model"
)

type podcastEpisodeReader struct {
	cacheKey
	a       *artwork
	episode model.PodcastEpisode
}

func newPodcastEpisodeArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID) (*podcastEpisodeReader, error) {
	pe, err := artwork.ds.PodcastEpisode(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}

	a := &podcastEpisodeReader{
		a:       artwork,
		episode: *pe,
	}

	a.cacheKey.artID = artID
	return a, nil
}

func (p *podcastEpisodeReader) LastUpdated() time.Time {
	return p.lastUpdate
}

func (p *podcastEpisodeReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	if p.episode.ImageUrl == "" {
		return nil, "", nil
	}

	imageUrl, err := url.Parse(p.episode.ImageUrl)
	if err != nil {
		return nil, "", err
	}

	return fromURL(ctx, imageUrl)
}

var _ artworkReader = (*podcastEpisodeReader)(nil)
