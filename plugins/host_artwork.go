package plugins

import (
	"context"

	"github.com/navidrome/navidrome/core/publicurl"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
)

type artworkServiceImpl struct{}

func newArtworkService() host.ArtworkService {
	return &artworkServiceImpl{}
}

func (a *artworkServiceImpl) GetArtistUrl(ctx context.Context, id string, size int32) (string, error) {
	artID := model.ArtworkID{Kind: model.KindArtistArtwork, ID: id}
	return publicurl.ImageURL(ctx, artID, int(size)), nil
}

func (a *artworkServiceImpl) GetAlbumUrl(ctx context.Context, id string, size int32) (string, error) {
	artID := model.ArtworkID{Kind: model.KindAlbumArtwork, ID: id}
	return publicurl.ImageURL(ctx, artID, int(size)), nil
}

func (a *artworkServiceImpl) GetTrackUrl(ctx context.Context, id string, size int32) (string, error) {
	artID := model.ArtworkID{Kind: model.KindMediaFileArtwork, ID: id}
	return publicurl.ImageURL(ctx, artID, int(size)), nil
}

func (a *artworkServiceImpl) GetPlaylistUrl(ctx context.Context, id string, size int32) (string, error) {
	artID := model.ArtworkID{Kind: model.KindPlaylistArtwork, ID: id}
	return publicurl.ImageURL(ctx, artID, int(size)), nil
}

var _ host.ArtworkService = (*artworkServiceImpl)(nil)
