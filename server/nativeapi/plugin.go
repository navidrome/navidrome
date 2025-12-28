package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins"
	"github.com/navidrome/navidrome/server"
)

func (api *Router) addPluginRoute(r chi.Router) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.ds.Plugin(ctx)
	}

	r.Route("/plugin", func(r chi.Router) {
		r.Use(pluginsEnabledMiddleware)
		r.Get("/", rest.GetAll(constructor))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			r.Put("/", api.updatePlugin)
		})
	})
}

// Middleware to check if plugins feature is enabled
func pluginsEnabledMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !conf.Server.Plugins.Enabled {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// PluginUpdateRequest represents the fields that can be updated via the API
type PluginUpdateRequest struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Config  *string `json:"config,omitempty"`
}

func (api *Router) updatePlugin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()
	repo := api.ds.Plugin(ctx)
	manager := plugins.GetManager()

	// Get existing plugin to verify it exists
	plugin, err := repo.Get(id)
	if err != nil {
		if errors.Is(err, rest.ErrPermissionDenied) {
			http.Error(w, "Access denied: admin privileges required", http.StatusForbidden)
			return
		}
		if errors.Is(err, model.ErrNotFound) {
			http.Error(w, "Plugin not found", http.StatusNotFound)
			return
		}
		log.Error(ctx, "Error getting plugin", "id", id, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Parse update request
	var req PluginUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, "Error decoding request", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// If manager is configured, use it to properly load/unload plugins
	// Otherwise, fall back to direct DB operations (e.g., in tests)
	if manager.IsConfigured() {
		// Handle config update first (if provided)
		if req.Config != nil {
			// Validate JSON if not empty
			if *req.Config != "" && !isValidJSON(*req.Config) {
				http.Error(w, "Invalid JSON in config field", http.StatusBadRequest)
				return
			}
			if err := manager.UpdatePluginConfig(ctx, id, *req.Config); err != nil {
				log.Error(ctx, "Error updating plugin config", "id", id, err)
				http.Error(w, "Error updating plugin configuration: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Handle enable/disable
		if req.Enabled != nil {
			if *req.Enabled {
				if err := manager.EnablePlugin(ctx, id); err != nil {
					log.Error(ctx, "Error enabling plugin", "id", id, err)
					// Refresh plugin from DB to get the error
					plugin, _ = repo.Get(id)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnprocessableEntity)
					_ = json.NewEncoder(w).Encode(plugin)
					return
				}
			} else {
				if err := manager.DisablePlugin(ctx, id); err != nil {
					log.Error(ctx, "Error disabling plugin", "id", id, err)
					http.Error(w, "Error disabling plugin: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	} else {
		// Fallback: direct DB operations (for tests or when manager is not started)
		if req.Config != nil {
			if *req.Config != "" && !isValidJSON(*req.Config) {
				http.Error(w, "Invalid JSON in config field", http.StatusBadRequest)
				return
			}
			plugin.Config = *req.Config
		}
		if req.Enabled != nil {
			plugin.Enabled = *req.Enabled
		}
		if err := repo.Put(plugin); err != nil {
			if errors.Is(err, rest.ErrPermissionDenied) {
				http.Error(w, "Access denied: admin privileges required", http.StatusForbidden)
				return
			}
			log.Error(ctx, "Error updating plugin", "id", id, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Refresh and return updated plugin
	plugin, err = repo.Get(id)
	if err != nil {
		log.Error(ctx, "Error getting updated plugin", "id", id, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(plugin); err != nil {
		log.Error(ctx, "Error encoding plugin response", err)
	}
}

// isValidJSON checks if a string is valid JSON
func isValidJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}
