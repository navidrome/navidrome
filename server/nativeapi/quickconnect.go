package nativeapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/quickconnect"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

type quickConnectRequest struct {
	AppName    string `json:"appName"`
	AppVersion string `json:"appVersion"`
	DeviceName string `json:"deviceName"`
}

func (api *Router) addQuickConnectRoute(r chi.Router) {
	if !quickconnect.Enabled() {
		return
	}
	r.Route("/quickconnect", func(r chi.Router) {
		r.Get("/", lookupQuickConnect(api.quickConnect))
		r.Post("/authorize", authorizeQuickConnect(api.quickConnect))
	})
}

func lookupQuickConnect(qc quickconnect.QuickConnect) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := qc.Lookup(r.URL.Query().Get("code"))
		writeQuickConnectResult(w, r, req, err)
	}
}

func authorizeQuickConnect(qc quickconnect.QuickConnect) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Code string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)
		req, err := qc.Authorize(body.Code, user.ID)
		if err == nil {
			log.Info(ctx, "Quick Connect sign-in approved", "username", user.UserName, "client", req.App, "device", req.Name)
		}
		writeQuickConnectResult(w, r, req, err)
	}
}

func writeQuickConnectResult(w http.ResponseWriter, r *http.Request, req quickconnect.Request, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		http.Error(w, "Unknown code", http.StatusNotFound)
	case errors.Is(err, quickconnect.ErrAlreadyAuthorized):
		http.Error(w, "Code already used", http.StatusConflict)
	case err != nil:
		log.Error(r.Context(), "Quick Connect failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	default:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(quickConnectRequest{AppName: req.App, AppVersion: req.AppVersion, DeviceName: req.Name})
	}
}
