package plugins

import (
	"context"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/plugins/api"
)

type wasmAlbumAgent struct {
	*wasmBasePlugin[api.AlbumMetadataService]
}

func (w *wasmAlbumAgent) AgentName() string {
	return w.name
}

func (w *wasmAlbumAgent) error(err error) error {
	if err != nil && (err.Error() == api.ErrNotFound.Error() || err.Error() == api.ErrNotImplemented.Error()) {
		return agents.ErrNotFound
	}
	return err
}

func isAlbumNotFound(err error) bool {
	return err != nil && (err.Error() == api.ErrNotFound.Error() || err.Error() == api.ErrNotImplemented.Error())
}

func callAlbumMethod[R any](ctx context.Context, w *wasmAlbumAgent, methodName string, fn func(inst api.AlbumMetadataService) (R, error)) (R, error) {
	inst, done, err := w.getInstance(ctx, methodName, isAlbumNotFound)
	var r R
	if err != nil {
		return r, err
	}
	defer func() { done(err) }()
	r, err = fn(inst)
	return r, w.error(err)
}

// AlbumMetadataService methods
func (w *wasmAlbumAgent) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*agents.AlbumInfo, error) {
	return callAlbumMethod(ctx, w, "GetAlbumInfo", func(inst api.AlbumMetadataService) (*agents.AlbumInfo, error) {
		res, err := inst.GetAlbumInfo(ctx, &api.AlbumInfoRequest{Name: name, Artist: artist, Mbid: mbid})
		if err != nil {
			return nil, err
		}
		if res == nil || res.Info == nil {
			return nil, agents.ErrNotFound
		}
		info := res.Info
		return &agents.AlbumInfo{
			Name:        info.Name,
			MBID:        info.Mbid,
			Description: info.Description,
			URL:         info.Url,
		}, nil
	})
}

func (w *wasmAlbumAgent) GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]agents.ExternalImage, error) {
	return callAlbumMethod(ctx, w, "GetAlbumImages", func(inst api.AlbumMetadataService) ([]agents.ExternalImage, error) {
		res, err := inst.GetAlbumImages(ctx, &api.AlbumImagesRequest{Name: name, Artist: artist, Mbid: mbid})
		if err != nil {
			return nil, err
		}
		return convertExternalImages(res.Images), nil
	})
}

func convertExternalImages(images []*api.ExternalImage) []agents.ExternalImage {
	result := make([]agents.ExternalImage, 0, len(images))
	for _, img := range images {
		result = append(result, agents.ExternalImage{
			URL:  img.GetUrl(),
			Size: int(img.GetSize()),
		})
	}
	return result
}
