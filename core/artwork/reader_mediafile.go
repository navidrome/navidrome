package artwork

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
)

type mediafileArtworkReader struct {
	cacheKey
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
	_, _, imagesUpdatedAt, err := loadAlbumFoldersPaths(ctx, artwork.ds, *al)
	if err != nil {
		return nil, err
	}
	a := &mediafileArtworkReader{
		a:         artwork,
		mediafile: *mf,
		album:     *al,
	}
	a.cacheKey.artID = artID
	a.cacheKey.lastUpdate = mf.UpdatedAt
	if al.UpdatedAt.After(a.cacheKey.lastUpdate) {
		a.cacheKey.lastUpdate = al.UpdatedAt
	}
	if imagesUpdatedAt != nil && imagesUpdatedAt.After(a.cacheKey.lastUpdate) {
		a.cacheKey.lastUpdate = *imagesUpdatedAt
	}
	return a, nil
}

func (a *mediafileArtworkReader) Key() string {
	return fmt.Sprintf(
		"%s.%t",
		a.cacheKey.Key(),
		conf.Server.EnableMediaFileCoverArt,
	)
}
func (a *mediafileArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *mediafileArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	var ff []sourceFunc
	if a.mediafile.CoverArtID().Kind == model.KindMediaFileArtwork {
		path := a.mediafile.AbsolutePath()
		ff = []sourceFunc{
			fromTag(ctx, path),
			fromFFmpegTag(ctx, a.a.ffmpeg, path),
		}
	}
	// For multi-disc albums, fall back to disc artwork first; for single-disc albums,
	// skip disc resolution (it would just fall through to album art anyway).
	if len(a.album.Discs) > 1 {
		ff = append(ff, fromAlbum(ctx, a.a, a.mediafile.DiscCoverArtID()))
	} else {
		ff = append(ff, fromAlbum(ctx, a.a, a.mediafile.AlbumCoverArtID()))
	}
	return selectImageReader(ctx, a.artID, ff...)
}
