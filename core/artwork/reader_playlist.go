package artwork

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"io"
	"time"

	"github.com/disintegration/imaging"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

type playlistArtworkReader struct {
	cacheKey
	a  *artwork
	pl model.Playlist
}

const tileSize = 600

func newPlaylistArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID) (*playlistArtworkReader, error) {
	pl, err := artwork.ds.Playlist(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	a := &playlistArtworkReader{
		a:  artwork,
		pl: *pl,
	}
	a.cacheKey.artID = artID
	a.cacheKey.lastUpdate = pl.UpdatedAt
	return a, nil
}

func (a *playlistArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *playlistArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	ff := []sourceFunc{
		a.fromGeneratedTiledCover(ctx),
		fromAlbumPlaceholder(),
	}
	return selectImageReader(ctx, a.artID, ff...)
}

func (a *playlistArtworkReader) fromGeneratedTiledCover(ctx context.Context) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		tiles, err := a.loadTiles(ctx)
		if err != nil {
			return nil, "", err
		}
		r, err := a.createTiledImage(ctx, tiles)
		return r, "", err
	}
}

func toArtworkIDs(albumIDs []string) []model.ArtworkID {
	return slice.Map(albumIDs, func(id string) model.ArtworkID {
		al := model.Album{ID: id}
		return al.CoverArtID()
	})
}

func (a *playlistArtworkReader) loadTiles(ctx context.Context) ([]image.Image, error) {
	tracksRepo := a.a.ds.Playlist(ctx).Tracks(a.pl.ID, false)
	albumIds, err := tracksRepo.GetAlbumIDs(model.QueryOptions{Max: 4, Sort: "random()"})
	if err != nil {
		log.Error(ctx, "Error getting album IDs for playlist", "id", a.pl.ID, "name", a.pl.Name, err)
		return nil, err
	}
	ids := toArtworkIDs(albumIds)

	var tiles []image.Image
	for len(tiles) < 4 {
		if len(ids) == 0 {
			break
		}
		id := ids[len(ids)-1]
		ids = ids[0 : len(ids)-1]
		r, _, err := fromAlbum(ctx, a.a, id)()
		if err != nil {
			continue
		}
		tile, err := a.createTile(ctx, r)
		if err == nil {
			tiles = append(tiles, tile)
		}
		_ = r.Close()
	}
	switch len(tiles) {
	case 0:
		return nil, errors.New("could not find any eligible cover")
	case 2:
		tiles = append(tiles, tiles[1], tiles[0])
	case 3:
		tiles = append(tiles, tiles[0])
	}
	return tiles, nil
}

func (a *playlistArtworkReader) createTile(_ context.Context, r io.ReadCloser) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return imaging.Fill(img, tileSize/2, tileSize/2, imaging.Center, imaging.Lanczos), nil
}

func (a *playlistArtworkReader) createTiledImage(_ context.Context, tiles []image.Image) (io.ReadCloser, error) {
	buf := new(bytes.Buffer)
	var rgba draw.Image
	var err error
	if len(tiles) == 4 {
		rgba = image.NewRGBA(image.Rectangle{Max: image.Point{X: tileSize - 1, Y: tileSize - 1}})
		draw.Draw(rgba, rect(0), tiles[0], image.Point{}, draw.Src)
		draw.Draw(rgba, rect(1), tiles[1], image.Point{}, draw.Src)
		draw.Draw(rgba, rect(2), tiles[2], image.Point{}, draw.Src)
		draw.Draw(rgba, rect(3), tiles[3], image.Point{}, draw.Src)
		err = png.Encode(buf, rgba)
	} else {
		err = png.Encode(buf, tiles[0])
	}
	if err != nil {
		return nil, err
	}
	return io.NopCloser(buf), nil
}

func rect(pos int) image.Rectangle {
	r := image.Rectangle{}
	switch pos {
	case 1:
		r.Min.X = tileSize / 2
	case 2:
		r.Min.Y = tileSize / 2
	case 3:
		r.Min.X = tileSize / 2
		r.Min.Y = tileSize / 2
	}
	r.Max.X = r.Min.X + tileSize/2
	r.Max.Y = r.Min.Y + tileSize/2
	return r
}
