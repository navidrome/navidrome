package plugins

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
)

type artworkServiceImpl struct{}

func newArtworkService() host.ArtworkService {
	return &artworkServiceImpl{}
}

func (a *artworkServiceImpl) GetArtistUrl(_ context.Context, id string, size int32) (string, error) {
	artID := model.ArtworkID{Kind: model.KindArtistArtwork, ID: id}
	return a.imageURL(artID, int(size)), nil
}

func (a *artworkServiceImpl) GetAlbumUrl(_ context.Context, id string, size int32) (string, error) {
	artID := model.ArtworkID{Kind: model.KindAlbumArtwork, ID: id}
	return a.imageURL(artID, int(size)), nil
}

func (a *artworkServiceImpl) GetTrackUrl(_ context.Context, id string, size int32) (string, error) {
	artID := model.ArtworkID{Kind: model.KindMediaFileArtwork, ID: id}
	return a.imageURL(artID, int(size)), nil
}

func (a *artworkServiceImpl) GetPlaylistUrl(_ context.Context, id string, size int32) (string, error) {
	artID := model.ArtworkID{Kind: model.KindPlaylistArtwork, ID: id}
	return a.imageURL(artID, int(size)), nil
}

// imageURL generates a public URL for artwork, replicating the logic from server/public
// to avoid import cycles.
func (a *artworkServiceImpl) imageURL(artID model.ArtworkID, size int) string {
	token, _ := auth.CreatePublicToken(map[string]any{"id": artID.String()})
	uri := path.Join(consts.URLPathPublicImages, token)
	params := url.Values{}
	if size > 0 {
		params.Add("size", strconv.Itoa(size))
	}
	return a.publicURL(uri, params)
}

// publicURL builds the full URL using ShareURL config or falling back to localhost.
func (a *artworkServiceImpl) publicURL(u string, params url.Values) string {
	var scheme, host string
	if conf.Server.ShareURL != "" {
		shareURL, _ := url.Parse(conf.Server.ShareURL)
		scheme = shareURL.Scheme
		host = shareURL.Host
	} else {
		scheme = "http"
		host = "localhost"
	}
	buildURL, _ := url.Parse(u)
	buildURL.Scheme = scheme
	buildURL.Host = host
	if len(params) > 0 {
		buildURL.RawQuery = params.Encode()
	}
	return buildURL.String()
}

// createRequest creates a dummy HTTP request for URL generation.
// Kept for reference but no longer used after refactoring.
func (a *artworkServiceImpl) createRequest() *http.Request {
	var scheme, host string
	if conf.Server.ShareURL != "" {
		shareURL, _ := url.Parse(conf.Server.ShareURL)
		scheme = shareURL.Scheme
		host = shareURL.Host
	} else {
		scheme = "http"
		host = "localhost"
	}
	r, _ := http.NewRequest("GET", fmt.Sprintf("%s://%s", scheme, host), nil)
	return r
}

var _ host.ArtworkService = (*artworkServiceImpl)(nil)
