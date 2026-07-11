package nativeapi

import (
	"encoding/json"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/ldapauth"
)

func (api *Router) addLDAPRoute(r chi.Router) {
	r.Route("/ldap", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			cfg, err := ldapauth.NewStore().Load(r.Context())
			if err != nil {
				_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
				return
			}
			_ = rest.RespondWithJSON(w, http.StatusOK, map[string]any{"id": "ldap", "sources": cfg.Sources})
		})
		r.Put("/", func(w http.ResponseWriter, r *http.Request) {
			var cfg ldapauth.Config
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				_ = rest.RespondWithError(w, http.StatusUnprocessableEntity, err.Error())
				return
			}
			if err := ldapauth.NewStore().Save(r.Context(), cfg); err != nil {
				_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
				return
			}
			_ = rest.RespondWithJSON(w, http.StatusOK, map[string]any{"id": "ldap", "sources": cfg.Sources})
		})
		r.Post("/test", func(w http.ResponseWriter, r *http.Request) {
			var src ldapauth.Source
			if err := json.NewDecoder(r.Body).Decode(&src); err != nil {
				_ = rest.RespondWithError(w, http.StatusUnprocessableEntity, err.Error())
				return
			}
			src, err := ldapauth.TestAndCache(r.Context(), src)
			if err != nil {
				_ = rest.RespondWithError(w, http.StatusBadGateway, err.Error())
				return
			}
			_ = rest.RespondWithJSON(w, http.StatusOK, src)
		})
	})
}
