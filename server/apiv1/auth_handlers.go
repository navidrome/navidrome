package apiv1

import (
	"cmp"
	"context"

	"github.com/navidrome/navidrome/core/apiauth"
	"github.com/navidrome/navidrome/utils/gg"
)

const defaultPageSize = 100

func (rt *Router) CreateAccessToken(ctx context.Context, req CreateAccessTokenRequestObject) (CreateAccessTokenResponseObject, error) {
	p, err := principal(apiauth.PrincipalFrom(ctx))
	if err != nil {
		return nil, err
	}
	var requested []string
	if req.Body != nil {
		requested = fromScopeRequests(req.Body.Scopes)
	}
	tok, err := rt.auth.Mint(ctx, p, requested)
	if err != nil {
		return nil, err
	}
	return CreateAccessToken200JSONResponse{
		AccessToken: tok.Token,
		TokenType:   AccessTokenTokenTypeBearer,
		ExpiresIn:   int(tok.ExpiresIn.Seconds()),
		Scopes:      toScopes(tok.Scopes),
	}, nil
}

func (rt *Router) ListGrants(ctx context.Context, req ListGrantsRequestObject) (ListGrantsResponseObject, error) {
	p, err := principal(apiauth.PrincipalFrom(ctx))
	if err != nil {
		return nil, err
	}
	offset := gg.V(req.Params.OffsetParam)
	limit := cmp.Or(gg.V(req.Params.LimitParam), defaultPageSize)
	grants, total, err := rt.auth.ListGrants(ctx, p, offset, limit)
	if err != nil {
		return nil, err
	}
	items := make([]Grant, len(grants))
	for i, g := range grants {
		items[i] = toGrant(g, p.GrantID)
	}
	return ListGrants200JSONResponse{Items: items, Total: int(total), Offset: offset, Limit: limit}, nil
}

func (rt *Router) RevokeGrant(ctx context.Context, req RevokeGrantRequestObject) (RevokeGrantResponseObject, error) {
	p, err := principal(apiauth.PrincipalFrom(ctx))
	if err != nil {
		return nil, err
	}
	if err := rt.auth.RevokeGrant(ctx, p, req.Id); err != nil {
		return nil, err
	}
	return RevokeGrant204Response{}, nil
}

func (rt *Router) Logout(ctx context.Context, _ LogoutRequestObject) (LogoutResponseObject, error) {
	p, err := principal(apiauth.PrincipalFrom(ctx))
	if err != nil {
		return nil, err
	}
	if err := rt.auth.Logout(ctx, p); err != nil {
		return nil, err
	}
	return Logout200JSONResponse{LogoutUrl: nil}, nil
}
