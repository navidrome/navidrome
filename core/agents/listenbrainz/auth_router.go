package listenbrainz

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
)

type sessionKeysRepo interface {
	Put(ctx context.Context, userId, sessionKey string) error
	Get(ctx context.Context, userId string) (string, error)
	Delete(ctx context.Context, userId string) error
}

type Router struct {
	http.Handler
	ds          model.DataStore
	sessionKeys sessionKeysRepo
	client      *client
}

func NewRouter(ds model.DataStore) *Router {
	r := &Router{
		ds:          ds,
		sessionKeys: &agents.SessionKeys{DataStore: ds, KeyName: sessionKeyProperty},
	}
	r.Handler = r.routes()
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	r.client = newClient(conf.Server.ListenBrainz.BaseURL, hc)
	return r
}

func (s *Router) routes() http.Handler {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(server.Authenticator(s.ds))
		r.Use(server.JWTRefresher)

		r.Get("/link", s.getLinkStatus)
		r.Put("/link", s.link)
		r.Delete("/link", s.unlink)
	})

	return r
}

func (s *Router) getLinkStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{}
	u, _ := request.UserFrom(r.Context())
	key, err := s.sessionKeys.Get(r.Context(), u.ID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		resp["error"] = err
		resp["status"] = false
		_ = rest.RespondWithJSON(w, http.StatusInternalServerError, resp)
		return
	}
	resp["status"] = key != ""
	_ = rest.RespondWithJSON(w, http.StatusOK, resp)
}

func (s *Router) link(w http.ResponseWriter, r *http.Request) {
	type tokenPayload struct {
		Token string `json:"token"`
	}
	var payload tokenPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if payload.Token == "" {
		_ = rest.RespondWithError(w, http.StatusBadRequest, "Token is required")
		return
	}

	u, _ := request.UserFrom(r.Context())
	resp, err := s.client.validateToken(r.Context(), payload.Token)
	if err != nil {
		log.Error(r.Context(), "Could not validate ListenBrainz token", "userId", u.ID, "requestId", middleware.GetReqID(r.Context()), err)
		_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !resp.Valid {
		_ = rest.RespondWithError(w, http.StatusBadRequest, "Invalid token")
		return
	}

	err = s.sessionKeys.Put(r.Context(), u.ID, payload.Token)
	if err != nil {
		log.Error("Could not save ListenBrainz token", "userId", u.ID, "requestId", middleware.GetReqID(r.Context()), err)
		_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = rest.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"status": resp.Valid, "user": resp.UserName})
}

func (s *Router) unlink(w http.ResponseWriter, r *http.Request) {
	u, _ := request.UserFrom(r.Context())
	err := s.sessionKeys.Delete(r.Context(), u.ID)
	if err != nil {
		_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
	} else {
		_ = rest.RespondWithJSON(w, http.StatusOK, map[string]string{})
	}
}
