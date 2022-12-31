package artwork

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
)

type artistReader struct {
	cacheKey
	a      *artwork
	artist model.Artist
	files  string
}

func newArtistReader(ctx context.Context, artwork *artwork, artID model.ArtworkID) (*artistReader, error) {
	ar, err := artwork.ds.Artist(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	als, err := artwork.ds.Album(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_artist_id": artID.ID}})
	if err != nil {
		return nil, err
	}
	a := &artistReader{
		a:      artwork,
		artist: *ar,
	}
	a.cacheKey.lastUpdate = ar.ExternalInfoUpdatedAt
	var files []string
	for _, al := range als {
		files = append(files, al.ImageFiles)
		if a.cacheKey.lastUpdate.Before(al.UpdatedAt) {
			a.cacheKey.lastUpdate = al.UpdatedAt
		}
	}
	a.files = strings.Join(files, string(filepath.ListSeparator))
	a.cacheKey.artID = artID
	return a, nil
}

func (a *artistReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *artistReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	return selectImageReader(ctx, a.artID,
		fromExternalFile(ctx, a.files, "artist.*"),
		fromExternalSource(ctx, a.artist),
		fromArtistPlaceholder(),
	)
}

func fromExternalSource(ctx context.Context, ar model.Artist) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		imageUrl := ar.ArtistImageUrl()
		if !strings.HasPrefix(imageUrl, "http") {
			return nil, "", nil
		}
		hc := http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, imageUrl, nil)
		resp, err := hc.Do(req)
		if err != nil {
			return nil, "", err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, "", fmt.Errorf("error retrieveing cover from %s: %s", imageUrl, resp.Status)
		}
		return resp.Body, imageUrl, nil
	}
}
