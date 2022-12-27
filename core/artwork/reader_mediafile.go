package artwork

import (
	"context"
	"io"
	"time"

	"github.com/navidrome/navidrome/model"
)

type mediafileArtworkReader struct {
	cacheItem
	a         *artwork
	mediafile model.MediaFile
	album     model.Album
}

func newMediafileArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID) (*mediafileArtworkReader, error) {
	mf, err := artwork.ds.MediaFile(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	al, err := artwork.ds.Album(ctx).Get(mf.AlbumID)
	if err != nil {
		return nil, err
	}
	a := &mediafileArtworkReader{
		a:         artwork,
		mediafile: *mf,
		album:     *al,
	}
	a.cacheItem.artID = artID
	a.cacheItem.lastUpdate = a.LastUpdated()
	return a, nil
}

func (a *mediafileArtworkReader) LastUpdated() time.Time {
	if a.album.UpdatedAt.After(a.mediafile.UpdatedAt) {
		return a.album.UpdatedAt
	}
	return a.mediafile.UpdatedAt
}

func (a *mediafileArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	var ff []sourceFunc
	if a.mediafile.CoverArtID().Kind == model.KindMediaFileArtwork {
		ff = []sourceFunc{
			fromTag(a.mediafile.Path),
			fromFFmpegTag(ctx, a.a.ffmpeg, a.mediafile.Path),
		}
	}
	ff = append(ff, fromAlbum(ctx, a.a, a.mediafile.AlbumCoverArtID()))
	r, source := extractImage(ctx, a.artID, ff...)
	return r, source, nil
}

func fromAlbum(ctx context.Context, a *artwork, id model.ArtworkID) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		r, err := a.Get(ctx, id.String(), 0)
		if err != nil {
			return nil, "", err
		}
		return r, id.String(), nil
	}
}
