package apiv1

import (
	"cmp"
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/slice"
)

const defaultPageSize = 100

func (rt *Router) ListGrants(ctx context.Context, req ListGrantsRequestObject) (ListGrantsResponseObject, error) {
	p, err := principalFrom(ctx)
	if err != nil {
		return nil, err
	}
	offset := gg.V(req.Params.OffsetParam)
	limit := cmp.Or(gg.V(req.Params.LimitParam), defaultPageSize)
	grants, total, err := rt.auth.ListGrants(ctx, p, offset, limit)
	if err != nil {
		return nil, err
	}
	items := slice.Map(grants, func(g model.Grant) Grant { return toGrant(g, p.GrantID) })
	return ListGrants200JSONResponse{Items: items, Total: int(total), Offset: offset, Limit: limit}, nil
}

func (rt *Router) RevokeGrant(ctx context.Context, req RevokeGrantRequestObject) (RevokeGrantResponseObject, error) {
	p, err := principalFrom(ctx)
	if err != nil {
		return nil, err
	}
	if err := rt.auth.RevokeGrant(ctx, p, req.Id); err != nil {
		return nil, err
	}
	return RevokeGrant204Response{}, nil
}

func (rt *Router) Logout(ctx context.Context, _ LogoutRequestObject) (LogoutResponseObject, error) {
	p, err := principalFrom(ctx)
	if err != nil {
		return nil, err
	}
	if err := rt.auth.Logout(ctx, p); err != nil {
		return nil, err
	}
	return Logout200JSONResponse{LogoutUrl: nil}, nil
}
