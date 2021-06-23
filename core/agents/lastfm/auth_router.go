package lastfm

import (
	"bytes"
	"context"
	_ "embed"
	"net/http"
	"time"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/utils"
)

//go:embed token_received.html
var tokenReceivedPage []byte

type Router struct {
	http.Handler
	ds          model.DataStore
	sessionKeys *sessionKeys
	client      *Client
	apiKey      string
	secret      string
}

func NewRouter(ds model.DataStore) *Router {
	r := &Router{
		ds:          ds,
		apiKey:      conf.Server.LastFM.ApiKey,
		secret:      conf.Server.LastFM.Secret,
		sessionKeys: &sessionKeys{ds: ds},
	}
	r.Handler = r.routes()
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	r.client = NewClient(r.apiKey, r.secret, "en", hc)
	return r
}

func (s *Router) routes() http.Handler {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(server.Authenticator(s.ds))
		r.Use(server.JWTRefresher)

		r.Get("/link", s.getLinkStatus)
		r.Delete("/link", s.unlink)
	})

	r.Get("/link/callback", s.callback)

	return r
}

func (s *Router) getLinkStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{"status": true}
	key, err := s.sessionKeys.get(r.Context())
	if err != nil && err != model.ErrNotFound {
		resp["error"] = err
		resp["status"] = false
		_ = rest.RespondWithJSON(w, http.StatusInternalServerError, resp)
		return
	}
	resp["status"] = key != ""
	_ = rest.RespondWithJSON(w, http.StatusOK, resp)
}

func (s *Router) unlink(w http.ResponseWriter, r *http.Request) {
	err := s.sessionKeys.delete(r.Context())
	if err != nil {
		_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
	} else {
		_ = rest.RespondWithJSON(w, http.StatusOK, map[string]string{})
	}
}

func (s *Router) callback(w http.ResponseWriter, r *http.Request) {
	token := utils.ParamString(r, "token")
	if token == "" {
		_ = rest.RespondWithError(w, http.StatusBadRequest, "token not received")
		return
	}
	uid := utils.ParamString(r, "uid")
	if uid == "" {
		_ = rest.RespondWithError(w, http.StatusBadRequest, "uid not received")
		return
	}

	// Need to add user to context, as this is a non-authenticated endpoint, so it does not
	// automatically contain any user info
	ctx := request.WithUser(r.Context(), model.User{ID: uid})
	err := s.fetchSessionKey(ctx, uid, token)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("An error occurred while authorizing with Last.fm. \n\nRequest ID: " + middleware.GetReqID(ctx)))
		return
	}

	http.ServeContent(w, r, "response", time.Now(), bytes.NewReader(tokenReceivedPage))
}

func (s *Router) fetchSessionKey(ctx context.Context, uid, token string) error {
	sessionKey, err := s.client.GetSession(ctx, token)
	if err != nil {
		log.Error(ctx, "Could not fetch LastFM session key", "userId", uid, "token", token,
			"requestId", middleware.GetReqID(ctx), err)
		return err
	}
	err = s.sessionKeys.put(ctx, sessionKey)
	if err != nil {
		log.Error("Could not save LastFM session key", "userId", uid, "requestId", middleware.GetReqID(ctx), err)
	}
	return err
}
