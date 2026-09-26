package apiv1

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/core/apiauth"
)

// Login relies on model.ErrInvalidAuth mapping to a detail-less 401, so unknown user and wrong password look the same.
func (rt *Router) Login(ctx context.Context, req LoginRequestObject) (LoginResponseObject, error) {
	b := *req.Body
	issued, err := rt.auth.Login(ctx, b.Username, b.Password, clientMeta(b), fromScopeRequests(b.Scopes))
	if err != nil {
		return nil, err
	}
	return Login200JSONResponse(toGrantCreated(issued)), nil
}

func (rt *Router) SetupFirstAdmin(ctx context.Context, req SetupFirstAdminRequestObject) (SetupFirstAdminResponseObject, error) {
	b := *req.Body
	issued, err := rt.auth.Setup(ctx, b.Username, b.Password, clientMeta(b), fromScopeRequests(b.Scopes))
	if err != nil {
		return nil, err
	}
	return SetupFirstAdmin201JSONResponse(toGrantCreated(issued)), nil
}

func (rt *Router) ChangePassword(ctx context.Context, req ChangePasswordRequestObject) (ChangePasswordResponseObject, error) {
	p, err := principal(apiauth.PrincipalFrom(ctx))
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
