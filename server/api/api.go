//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen -config ./openapi_api.cfg.yaml "../../api/spec.yml"
//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen -config ./openapi_types.cfg.yaml "../../api/spec.yml"

package api

import (
	"context"
	"net/http"

	middleware "github.com/deepmap/oapi-codegen/pkg/chi-middleware"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
)

var spec = func() *openapi3.T {
	s, _ := GetSwagger()
	//s.Servers = nil
	//s.Components.SecuritySchemes = nil
	s.Security = nil //TODO
	return s
}()

func New(ds model.DataStore) *Router {
	r := &Router{ds: ds}
	mux := chi.NewRouter()
	mux.Use(middleware.OapiRequestValidatorWithOptions(spec, &middleware.Options{
		ErrorHandler: validationErrorHandler,
	}))

	handler := NewStrictHandlerWithOptions(r, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  apiErrorHandler,
		ResponseErrorHandlerFunc: apiErrorHandler,
	})
	r.Handler = HandlerWithOptions(handler, ChiServerOptions{
		BaseRouter:  mux,
		Middlewares: []MiddlewareFunc{storeRequestInContext},
	})
	return r
}

var _ StrictServerInterface = (*Router)(nil)

type Router struct {
	http.Handler
	ds model.DataStore
}

func (a *Router) GetAlbums(ctx context.Context, request GetAlbumsRequestObject) (GetAlbumsResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (a *Router) GetAlbum(ctx context.Context, request GetAlbumRequestObject) (GetAlbumResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (a *Router) GetArtists(ctx context.Context, request GetArtistsRequestObject) (GetArtistsResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (a *Router) GetArtist(ctx context.Context, request GetArtistRequestObject) (GetArtistResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (a *Router) GetServerInfo(_ context.Context, _ GetServerInfoRequestObject) (GetServerInfoResponseObject, error) {
	return GetServerInfo200JSONResponse{
		Data: ServerInfo{
			AuthRequired:  true,
			Features:      []ServerInfoFeatures{},
			Server:        consts.AppName,
			ServerVersion: consts.Version,
		},
	}, nil
}

func (a *Router) GetTracks(ctx context.Context, request GetTracksRequestObject) (GetTracksResponseObject, error) {
	options := toQueryOptions(request.Params)
	mfs, err := a.ds.MediaFile(ctx).GetAll(options)
	if err != nil {
		return nil, err
	}
	cnt, err := a.ds.MediaFile(ctx).CountAll(options)
	if err != nil {
		return nil, err
	}
	baseUrl := baseResourceUrl(ctx, "tracks")
	links, meta := buildPaginationLinksAndMeta(int32(cnt), request.Params, baseUrl)
	return GetTracks200JSONResponse{
		Data:  toAPITracks(mfs),
		Links: links,
		Meta:  &meta,
	}, nil
}

func (a *Router) GetTrack(ctx context.Context, request GetTrackRequestObject) (GetTrackResponseObject, error) {
	mf, err := a.ds.MediaFile(ctx).Get(request.TrackId)
	if err != nil {
		return nil, err
	}
	return GetTrack200JSONResponse{Data: toAPITrack(*mf)}, nil
}
