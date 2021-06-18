package lastfm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"

	"github.com/ReneKroon/ttlcache/v2"

	"github.com/deluan/rest"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const (
	authURL                  = "https://www.last.fm/api/auth/"
	sessionKeyPropertyPrefix = "LastFMSessionKey_"
)

var (
	ErrLinkPending = errors.New("linking pending")
	ErrUnlinked    = errors.New("account not linked")
)

type Router struct {
	http.Handler
	ds         model.DataStore
	client     *Client
	sessionMan *sessionMan
	apiKey     string
	secret     string
}

func NewRouter(ds model.DataStore) *Router {
	r := &Router{ds: ds, apiKey: lastFMAPIKey, secret: lastFMAPISecret}
	r.Handler = r.routes()
	if conf.Server.LastFM.ApiKey != "" {
		r.apiKey = conf.Server.LastFM.ApiKey
		r.secret = conf.Server.LastFM.Secret
	}
	r.client = NewClient(r.apiKey, r.secret, "en", http.DefaultClient)
	r.sessionMan = newSessionMan(ds, r.client)
	return r
}

func (s *Router) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(server.Authenticator(s.ds))
	r.Use(server.JWTRefresher)

	r.Get("/link", s.starLink)
	r.Get("/link/status", s.getLinkStatus)
	r.Delete("/link", s.unlink)

	return r
}

func (s *Router) starLink(w http.ResponseWriter, r *http.Request) {
	token, err := s.client.GetToken(r.Context())
	if err != nil {
		log.Error(r.Context(), "Error obtaining token from LastFM", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("Error obtaining token from LastFM: %s", err)))
		return
	}
	username, _ := request.UsernameFrom(r.Context())
	s.sessionMan.FetchSession(username, token)
	params := url.Values{}
	params.Add("api_key", s.apiKey)
	params.Add("token", token)
	http.Redirect(w, r, authURL+"?"+params.Encode(), http.StatusFound)
}

func (s *Router) getLinkStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username, _ := request.UsernameFrom(ctx)
	_, err := s.sessionMan.Session(ctx, username)
	resp := map[string]string{"status": "linked"}
	if err != nil {
		switch err {
		case ErrLinkPending:
			resp["status"] = "pending"
		case ErrUnlinked:
			resp["status"] = "unlinked"
		default:
			resp["status"] = "unlinked"
			resp["error"] = err.Error()
			_ = rest.RespondWithJSON(w, http.StatusInternalServerError, resp)
			return
		}
	}
	_ = rest.RespondWithJSON(w, http.StatusOK, resp)
}

func (s *Router) unlink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username, _ := request.UsernameFrom(ctx)
	err := s.sessionMan.RemoveSession(ctx, username)
	if err != nil {
		_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
	} else {
		_ = rest.RespondWithJSON(w, http.StatusOK, map[string]string{})
	}
}

type sessionMan struct {
	ds     model.DataStore
	client *Client
	tokens *ttlcache.Cache
}

func newSessionMan(ds model.DataStore, client *Client) *sessionMan {
	s := &sessionMan{
		ds:     ds,
		client: client,
	}
	s.tokens = ttlcache.NewCache()
	s.tokens.SetCacheSizeLimit(0)
	_ = s.tokens.SetTTL(30 * time.Second)
	s.tokens.SkipTTLExtensionOnHit(true)
	go s.run()
	return s
}

func (s *sessionMan) FetchSession(username, token string) {
	_ = s.ds.Property(context.Background()).Delete(sessionKeyPropertyPrefix + username)
	_ = s.tokens.Set(username, token)
}

func (s *sessionMan) Session(ctx context.Context, username string) (string, error) {
	properties := s.ds.Property(context.Background())
	key, err := properties.Get(sessionKeyPropertyPrefix + username)
	if key != "" {
		return key, nil
	}
	if err != nil && err != model.ErrNotFound {
		return "", err
	}
	_, err = s.tokens.Get(username)
	if err == nil {
		return "", ErrLinkPending
	}
	return "", ErrUnlinked
}

func (s *sessionMan) RemoveSession(ctx context.Context, username string) error {
	_ = s.tokens.Remove(username)
	properties := s.ds.Property(context.Background())
	return properties.Delete(sessionKeyPropertyPrefix + username)
}

func (s *sessionMan) run() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		<-t.C
		if s.tokens.Count() == 0 {
			continue
		}
		s.fetchSessions()
	}
}

func (s *sessionMan) fetchSessions() {
	ctx := context.Background()
	for _, username := range s.tokens.GetKeys() {
		token, err := s.tokens.Get(username)
		if err != nil {
			log.Error("Error retrieving token from cache", "username", username, err)
			_ = s.tokens.Remove(username)
			continue
		}
		sessionKey, err := s.client.GetSession(ctx, token.(string))
		log.Debug(ctx, "Fetching session", "username", username, "sessionKey", sessionKey, "token", token, err)
		if err != nil {
			continue
		}
		properties := s.ds.Property(ctx)
		err = properties.Put(sessionKeyPropertyPrefix+username, sessionKey)
		if err != nil {
			log.Error("Could not save LastFM session key", "username", username, err)
		}
		_ = s.tokens.Remove(username)
	}
}
