package artwork

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/model"
)

type albumArtworkReader struct {
	cacheKey
	a     *artwork
	em    core.ExternalMetadata
	album model.Album
}

func newAlbumArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID, em core.ExternalMetadata) (*albumArtworkReader, error) {
	al, err := artwork.ds.Album(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	a := &albumArtworkReader{
		a:     artwork,
		em:    em,
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
	var ff = a.fromCoverArtPriority(ctx, a.a.ffmpeg, conf.Server.CoverArtPriority)
	ff = append(ff, fromAlbumPlaceholder())
	return selectImageReader(ctx, a.artID, ff...)
}

func (a *albumArtworkReader) fromCoverArtPriority(ctx context.Context, ffmpeg ffmpeg.FFmpeg, priority string) []sourceFunc {
	var ff []sourceFunc
	for _, pattern := range strings.Split(strings.ToLower(priority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "embedded":
			ff = append(ff, fromTag(a.album.EmbedArtPath), fromFFmpegTag(ctx, ffmpeg, a.album.EmbedArtPath))
		case pattern == "external":
			ff = append(ff, fromAlbumExternalSource(ctx, a.album, a.em))
		case a.album.ImageFiles != "":
			ff = append(ff, fromExternalFile(ctx, a.album.ImageFiles, pattern))
		}
	}
	return ff
}
