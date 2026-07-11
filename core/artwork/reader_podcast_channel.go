package artwork

import (
	"context"
	"io"
	"time"

	"github.com/navidrome/navidrome/model"
)

type podcastChannelArtworkReader struct {
	cacheKey
	a       *artwork
	channel model.PodcastChannel
}

func newPodcastChannelArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID) (*podcastChannelArtworkReader, error) {
	c, err := artwork.ds.PodcastChannel(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	a := &podcastChannelArtworkReader{a: artwork, channel: *c}
	a.cacheKey.artID = artID
	a.cacheKey.lastUpdate = c.UpdatedAt
	return a, nil
}

func (a *podcastChannelArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *podcastChannelArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	return selectImageReader(ctx, a.artID,
		a.fromPodcastChannelUploadedImage(),
	)
}

func (a *podcastChannelArtworkReader) fromPodcastChannelUploadedImage() sourceFunc {
	return fromLocalFile(a.channel.UploadedImagePath())
}
