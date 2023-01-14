package artwork

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/model"
)

type albumArtworkReader struct {
	cacheKey
	a     *artwork
	album model.Album
}

func newAlbumArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID) (*albumArtworkReader, error) {
	al, err := artwork.ds.Album(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	a := &albumArtworkReader{
		a:     artwork,
		album: *al,
	}
	a.cacheKey.artID = artID
	a.cacheKey.lastUpdate = al.UpdatedAt
	return a, nil
}

func (a *albumArtworkReader) LastUpdated() time.Time {
	return a.album.UpdatedAt
}

func (a *albumArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	var ff = fromCoverArtPriority(ctx, a.a.ffmpeg, conf.Server.CoverArtPriority, a.album)
	ff = append(ff, fromAlbumPlaceholder())
	return selectImageReader(ctx, a.artID, ff...)
}

func fromCoverArtPriority(ctx context.Context, ffmpeg ffmpeg.FFmpeg, priority string, al model.Album) []sourceFunc {
	var ff []sourceFunc
	for _, pattern := range strings.Split(strings.ToLower(priority), ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "embedded" {
			ff = append(ff, fromTag(al.EmbedArtPath), fromFFmpegTag(ctx, ffmpeg, al.EmbedArtPath))
			continue
		}
		if al.ImageFiles != "" {
			ff = append(ff, fromExternalFile(ctx, al.ImageFiles, pattern))
		}
	}
	return ff
}
