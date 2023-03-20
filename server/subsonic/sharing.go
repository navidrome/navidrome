package subsonic

import (
	"net/http"
	"strings"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/public"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils"
)

func (api *Router) GetShares(r *http.Request) (*responses.Subsonic, error) {
	repo := api.share.NewRepository(r.Context()).(model.ShareRepository)
	shares, err := repo.GetAll(model.QueryOptions{Sort: "created_at desc"})
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.Shares = &responses.Shares{}
	for _, share := range shares {
		response.Shares.Share = append(response.Shares.Share, api.buildShare(r, share))
	}
	return response, nil
}

func (api *Router) buildShare(r *http.Request, share model.Share) responses.Share {
	resp := responses.Share{
		ID:          share.ID,
		Url:         public.ShareURL(r, share.ID),
		Description: share.Description,
		Username:    share.Username,
		Created:     share.CreatedAt,
		Expires:     &share.ExpiresAt,
		LastVisited: share.LastVisitedAt,
		VisitCount:  int32(share.VisitCount),
	}
	if resp.Description == "" {
		resp.Description = share.Contents
	}
	if len(share.Albums) > 0 {
		resp.Entry = childrenFromAlbums(r.Context(), share.Albums)
	} else {
		resp.Entry = childrenFromMediaFiles(r.Context(), share.Tracks)
	}
	return resp
}

func (api *Router) CreateShare(r *http.Request) (*responses.Subsonic, error) {
	ids := utils.ParamStrings(r, "id")
	if len(ids) == 0 {
		return nil, newError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}

	description := utils.ParamString(r, "description")
	expires := utils.ParamTime(r, "expires", time.Time{})

	repo := api.share.NewRepository(r.Context())
	share := &model.Share{
		Description: description,
		ExpiresAt:   expires,
		ResourceIDs: strings.Join(ids, ","),
	}

	id, err := repo.(rest.Persistable).Save(share)
	if err != nil {
		return nil, err
	}

	share, err = repo.(model.ShareRepository).Get(id)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.Shares = &responses.Shares{Share: []responses.Share{api.buildShare(r, *share)}}
	return response, nil
}

func (api *Router) UpdateShare(r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	if id == "" {
		return nil, newError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}

	description := utils.ParamString(r, "description")
	expires := utils.ParamTime(r, "expires", time.Time{})

	repo := api.share.NewRepository(r.Context())
	share := &model.Share{
		ID:          id,
		Description: description,
		ExpiresAt:   expires,
	}

	err := repo.(rest.Persistable).Update(id, share)
	if err != nil {
		return nil, err
	}

	return newResponse(), nil
}

func (api *Router) DeleteShare(r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	if id == "" {
		return nil, newError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}

	repo := api.share.NewRepository(r.Context())
	err := repo.(rest.Persistable).Delete(id)
	if err != nil {
		return nil, err
	}

	return newResponse(), nil
}
