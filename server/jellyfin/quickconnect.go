package jellyfin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/quickconnect"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

func requireQuickConnect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !conf.Server.Jellyfin.QuickConnect {
			http.Error(w, "Quick connect is disabled", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (api *Router) quickConnectEnabled(w http.ResponseWriter, r *http.Request) {
	api.ok(w, r, conf.Server.Jellyfin.QuickConnect)
}

func (api *Router) quickConnectInitiate(w http.ResponseWriter, r *http.Request) {
	a := parseMediaBrowserAuth(r)
	if a.Client == "" || a.Device == "" || a.DeviceId == "" || a.Version == "" {
		http.Error(w, "Client, Device, DeviceId and Version are required", http.StatusBadRequest)
		return
	}
	req, err := api.quickConnect.Initiate(quickconnect.Device{
		ID: a.DeviceId, Name: a.Device, App: a.Client, AppVersion: a.Version,
	})
	if errors.Is(err, quickconnect.ErrTooManyRequests) {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, quickConnectResult(req))
}

func (api *Router) quickConnectConnect(w http.ResponseWriter, r *http.Request) {
	req, err := api.quickConnect.Status(r.URL.Query().Get("secret"))
	if err != nil {
		http.Error(w, "Unknown secret", http.StatusNotFound)
		return
	}
	api.ok(w, r, quickConnectResult(req))
}

func (api *Router) quickConnectAuthorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	caller, _ := request.UserFrom(ctx)
	userID := caller.ID
	if encoded := r.URL.Query().Get("userid"); encoded != "" {
		var ok bool
		if userID, ok = dto.DecodeID(encoded); !ok {
			http.Error(w, "Invalid userId", http.StatusBadRequest)
			return
		}
	}
	target := caller
	if userID != caller.ID {
		if !caller.IsAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		usr, err := api.ds.User(ctx).Get(userID)
		if errors.Is(err, model.ErrNotFound) {
			http.Error(w, "Unknown user", http.StatusNotFound)
			return
		}
		if err != nil {
			api.internalError(w, r, err)
			return
		}
		target = *usr
	}

	req, err := api.quickConnect.Authorize(r.URL.Query().Get("code"), target.ID)
	switch {
	case errors.Is(err, model.ErrNotFound):
		http.Error(w, "Unknown code", http.StatusNotFound)
	case errors.Is(err, quickconnect.ErrAlreadyAuthorized):
		http.Error(w, "Code already used", http.StatusConflict)
	case err != nil:
		api.internalError(w, r, err)
	default:
		log.Info(ctx, "Jellyfin API: Quick Connect sign-in approved", "username", target.UserName,
			"approvedBy", caller.UserName, "client", req.App, "device", req.Name)
		api.ok(w, r, true)
	}
}

type quickConnectDto struct {
	Secret string `json:"Secret"`
}

func (api *Router) authenticateWithQuickConnect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body quickConnectDto
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Secret == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	userID, err := api.quickConnect.Redeem(body.Secret)
	if err != nil {
		http.Error(w, "Unknown secret", http.StatusNotFound)
		return
	}
	usr, err := api.ds.User(ctx).Get(userID)
	if err != nil {
		log.Warn(ctx, "Jellyfin API: Quick Connect user not found", "userID", userID, err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	api.signIn(w, r, usr)
}

func quickConnectResult(req quickconnect.Request) dto.QuickConnectResult {
	return dto.QuickConnectResult{
		Authenticated: req.Authorized(),
		Secret:        req.Secret,
		Code:          req.Code,
		DeviceId:      req.ID,
		DeviceName:    req.Name,
		AppName:       req.App,
		AppVersion:    req.AppVersion,
		DateAdded:     dto.JellyfinDate(req.DateAdded),
	}
}
