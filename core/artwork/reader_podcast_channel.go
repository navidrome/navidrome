package artwork

import (
	"context"
	"io"
	"net/url"
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
		a.fromPodcastChannelExternalImage(ctx),
	)
}

func (a *podcastChannelArtworkReader) fromPodcastChannelUploadedImage() sourceFunc {
	return fromLocalFile(a.channel.UploadedImagePath())
}

// fromPodcastChannelExternalImage fetches the channel's artwork from the
// RSS feed's own image URL, so subscribed channels without an admin-uploaded
// override still show real cover art instead of a broken image.
func (a *podcastChannelArtworkReader) fromPodcastChannelExternalImage(ctx context.Context) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		if a.channel.CoverArtUrl == "" {
			return nil, "", ErrUnavailable
		}
		imageUrl, err := url.Parse(a.channel.CoverArtUrl)
		if err != nil {
			return nil, "", err
		}
		return fromURL(ctx, imageUrl)
	}
}
