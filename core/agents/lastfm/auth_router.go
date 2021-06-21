package lastfm

import (
	"bytes"
	"context"
	_ "embed"
	"net/http"
	"time"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
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
		ds:     ds,
		apiKey: conf.Server.LastFM.ApiKey,
		secret: conf.Server.LastFM.Secret,
	}
	r.sessionKeys = &sessionKeys{ds: ds}
	r.Handler = r.routes()
	r.client = NewClient(r.apiKey, r.secret, "en", http.DefaultClient)
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
	ctx := r.Context()
	u, _ := request.UserFrom(ctx)

	resp := map[string]interface{}{"status": true}
	key, err := s.sessionKeys.get(ctx, u.ID)
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
	ctx := r.Context()
	u, _ := request.UserFrom(ctx)

	err := s.sessionKeys.delete(ctx, u.ID)
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

	ctx := r.Context()
	err := s.fetchSessionKey(ctx, uid, token)
	if err != nil {
		_ = rest.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	http.ServeContent(w, r, "response", time.Now(), bytes.NewReader(tokenReceivedPage))
}

func (s *Router) fetchSessionKey(ctx context.Context, uid, token string) error {
	sessionKey, err := s.client.GetSession(ctx, token)
	if err != nil {
		log.Error(ctx, "Could not fetch LastFM session key", "userId", uid, "token", token, err)
		return err
	}
	err = s.sessionKeys.put(ctx, uid, sessionKey)
	if err != nil {
		log.Error("Could not save LastFM session key", "userId", uid, err)
	}
	return err
}

const (
	sessionKeyPropertyPrefix = "LastFMSessionKey_"
)

type sessionKeys struct {
	ds model.DataStore
}

func (sk *sessionKeys) put(ctx context.Context, uid string, sessionKey string) error {
	return sk.ds.Property(ctx).Put(sessionKeyPropertyPrefix+uid, sessionKey)
}

func (sk *sessionKeys) get(ctx context.Context, uid string) (string, error) {
	return sk.ds.Property(ctx).Get(sessionKeyPropertyPrefix + uid)
}

func (sk *sessionKeys) delete(ctx context.Context, uid string) error {
	return sk.ds.Property(ctx).Delete(sessionKeyPropertyPrefix + uid)
}
