package apiv1

import (
	"cmp"
	"context"
	"errors"

	"github.com/navidrome/navidrome/core/apiauth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/slice"
)

const defaultPageSize = 100

// Login relies on model.ErrInvalidAuth mapping to a detail-less 401, so unknown user and wrong password look the same.
func (rt *Router) Login(ctx context.Context, req LoginRequestObject) (LoginResponseObject, error) {
	b := *req.Body
	issued, err := rt.auth.Login(ctx, b.Username, b.Password, clientMeta(b), fromScopeRequests(b.Scopes))
	if err != nil {
		return nil, err
	}
	return Login200JSONResponse{Body: toGrantCreated(issued)}, nil
}

func (rt *Router) SetupFirstAdmin(ctx context.Context, req SetupFirstAdminRequestObject) (SetupFirstAdminResponseObject, error) {
	b := *req.Body
	issued, err := rt.auth.Setup(ctx, b.Username, b.Password, clientMeta(b), fromScopeRequests(b.Scopes))
	if err != nil {
		return nil, err
	}
	return SetupFirstAdmin201JSONResponse{Body: toGrantCreated(issued)}, nil
}

func (rt *Router) ChangePassword(ctx context.Context, req ChangePasswordRequestObject) (ChangePasswordResponseObject, error) {
	p, err := principalFrom(ctx)
	if err != nil {
		return nil, err
	}
	b := *req.Body
	revoke := true
	if b.RevokeOtherGrants != nil {
		revoke = *b.RevokeOtherGrants
	}
	err = rt.auth.ChangePassword(ctx, p, b.CurrentPassword, b.NewPassword, revoke)
	if errors.Is(err, apiauth.ErrCurrentPasswordMismatch) {
		return nil, validationFailed(ValidationError{Field: "currentPassword", Message: "is incorrect"})
	}
	if err != nil {
		return nil, err
	}
	return ChangePassword204Response{}, nil
}

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
