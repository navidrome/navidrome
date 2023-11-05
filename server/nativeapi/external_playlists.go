package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/external_playlists"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils"
)

var (
	errBadRange = errors.New("end must me greater than start")
)

type webError struct {
	Error string `json:"error"`
}

func requiredParamString(w *http.ResponseWriter, r *http.Request, param string) (string, bool) {
	p := utils.ParamString(r, param)
	if p == "" {
		replyError(r.Context(), *w, fmt.Errorf(`required param "%s" is missing`, param), http.StatusBadRequest)
		return p, false
	}
	return p, true
}

func replyError(ctx context.Context, w http.ResponseWriter, err error, status int) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	error := webError{Error: err.Error()}
	resp, _ := json.Marshal(error)
	_, writeErr := w.Write(resp)
	if writeErr != nil {
		log.Error(ctx, "Error sending json", "Error", err)
	}
}

func replyJson(ctx context.Context, w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	resp, _ := json.Marshal(data)
	_, err := w.Write(resp)

	if err != nil {
		log.Error(ctx, "Error sending json", "Error", err)
	}
}

func (n *Router) getAgents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(r.Context())

		agents := n.pls.GetAvailableAgents(ctx, user.ID)

		replyJson(ctx, w, agents)
	}
}

func (n *Router) getPlaylists() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(r.Context())

		start := utils.ParamInt(r, "_start", 0)
		end := utils.ParamInt(r, "_end", 0)

		if start >= end {
			replyError(ctx, w, errBadRange, http.StatusBadRequest)
			return
		}

		count := end - start

		agent, ok := requiredParamString(&w, r, "agent")
		if !ok {
			return
		}

		plsType, ok := requiredParamString(&w, r, "type")
		if !ok {
			return
		}

		lists, err := n.pls.GetPlaylists(ctx, start, count, user.ID, agent, plsType)

		if err != nil {
			replyError(ctx, w, err, http.StatusInternalServerError)
		} else {
			w.Header().Set("X-Total-Count", strconv.Itoa(lists.Total))

			replyJson(ctx, w, lists.Lists)
		}
	}
}

type externalImport struct {
	Agent     string                       `json:"agent"`
	Playlists external_playlists.ImportMap `json:"playlists"`
	Update    bool                         `json:"update"`
}

func (n *Router) fetchPlaylists() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(r.Context())

		defer r.Body.Close()

		data, err := io.ReadAll(r.Body)

		if err != nil {
			replyError(ctx, w, err, http.StatusBadRequest)
			return
		}

		var plsImport externalImport
		err = json.Unmarshal(data, &plsImport)

		if err != nil {
			replyError(ctx, w, err, http.StatusBadRequest)
			return
		}

		err = n.pls.ImportPlaylists(ctx, plsImport.Update, user.ID, plsImport.Agent, plsImport.Playlists)

		if err != nil {
			if errors.Is(model.ErrNotAuthorized, err) {
				replyError(ctx, w, err, http.StatusForbidden)
			} else {
				replyError(ctx, w, err, http.StatusInternalServerError)
			}
			return
		}

		replyJson(ctx, w, "")
	}
}

func (n *Router) syncPlaylist() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		plsId := chi.URLParam(r, "playlistId")

		err := n.pls.SyncPlaylist(ctx, plsId)

		if err != nil {
			log.Error(ctx, "Failed to sync playlist", "id", plsId, err)
			var code int

			if errors.Is(err, model.ErrNotAuthorized) {
				code = http.StatusForbidden
			} else if errors.Is(err, model.ErrNotFound) {
				code = http.StatusNotFound
			} else {
				code = http.StatusInternalServerError
			}

			replyError(ctx, w, err, code)
		} else {
			replyJson(ctx, w, "")
		}
	}
}

func (n *Router) externalPlaylistRoutes(r chi.Router) {
	r.Route("/externalPlaylist", func(r chi.Router) {
		r.Get("/", n.getPlaylists())
		r.Post("/", n.fetchPlaylists())
		r.Put("/sync/{playlistId}", n.syncPlaylist())

		r.Get("/agents", n.getAgents())
	})
}
