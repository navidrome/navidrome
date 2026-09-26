package apiv1

import "context"

func (rt *Router) GetCapabilities(context.Context, GetCapabilitiesRequestObject) (GetCapabilitiesResponseObject, error) {
	return GetCapabilities200JSONResponse{
		Core:     &CoreCapability{Version: 1},
		Password: &PasswordCapability{Version: 1},
	}, nil
}
