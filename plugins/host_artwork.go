package plugins

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host/artwork"
	"github.com/navidrome/navidrome/server/public"
)

type artworkServiceImpl struct{}

func (a *artworkServiceImpl) GetArtistUrl(_ context.Context, req *artwork.GetArtworkUrlRequest) (*artwork.GetArtworkUrlResponse, error) {
	artID := model.ArtworkID{Kind: model.KindArtistArtwork, ID: req.Id}
	imageURL := public.ImageURL(a.createRequest(), artID, int(req.Size))
	return &artwork.GetArtworkUrlResponse{Url: imageURL}, nil
}

func (a *artworkServiceImpl) GetAlbumUrl(_ context.Context, req *artwork.GetArtworkUrlRequest) (*artwork.GetArtworkUrlResponse, error) {
	artID := model.ArtworkID{Kind: model.KindAlbumArtwork, ID: req.Id}
	imageURL := public.ImageURL(a.createRequest(), artID, int(req.Size))
	return &artwork.GetArtworkUrlResponse{Url: imageURL}, nil
}

func (a *artworkServiceImpl) GetTrackUrl(_ context.Context, req *artwork.GetArtworkUrlRequest) (*artwork.GetArtworkUrlResponse, error) {
	artID := model.ArtworkID{Kind: model.KindMediaFileArtwork, ID: req.Id}
	imageURL := public.ImageURL(a.createRequest(), artID, int(req.Size))
	return &artwork.GetArtworkUrlResponse{Url: imageURL}, nil
}

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
