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
	"github.com/navidrome/navidrome/server"
)

func (api *Router) addPluginRoute(r chi.Router) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.ds.Plugin(ctx)
	}

	r.Route("/plugin", func(r chi.Router) {
		r.Use(pluginsEnabledMiddleware)
		r.Get("/", rest.GetAll(constructor))
		r.Post("/rescan", api.rescanPlugins)
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			r.Put("/", api.updatePlugin)
		})
	})
}

func (api *Router) rescanPlugins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := api.pluginManager.RescanPlugins(ctx); err != nil {
		log.Error(ctx, "Error rescanning plugins", err)
		http.Error(w, "Error rescanning plugins: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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
	Enabled      *bool   `json:"enabled,omitempty"`
	Config       *string `json:"config,omitempty"`
	Users        *string `json:"users,omitempty"`
	AllUsers     *bool   `json:"allUsers,omitempty"`
	Libraries    *string `json:"libraries,omitempty"`
	AllLibraries *bool   `json:"allLibraries,omitempty"`
}

func (api *Router) updatePlugin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()
	repo := api.ds.Plugin(ctx)

	// Get existing plugin to verify it exists
	if _, err := repo.Get(id); err != nil {
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

	// Handle config update (if provided)
	if req.Config != nil {
		if err := validateAndUpdateConfig(ctx, api.pluginManager, id, *req.Config, w); err != nil {
			log.Error(ctx, "Error updating plugin config", err)
			return
		}
	}

	// Handle users permission update (if provided)
	if req.Users != nil || req.AllUsers != nil {
		if err := validateAndUpdateUsers(ctx, api.pluginManager, repo, id, req, w); err != nil {
			log.Error(ctx, "Error updating plugin users", err)
			return
		}
	}

	// Handle libraries permission update (if provided)
	if req.Libraries != nil || req.AllLibraries != nil {
		if err := validateAndUpdateLibraries(ctx, api.pluginManager, repo, id, req, w); err != nil {
			log.Error(ctx, "Error updating plugin libraries", err)
			return
		}
	}

	// Handle enable/disable
	if req.Enabled != nil {
		if *req.Enabled {
			if enableErr := api.pluginManager.EnablePlugin(ctx, id); enableErr != nil {
				log.Error(ctx, "Error enabling plugin", "id", id, enableErr)
				// Refresh plugin from DB to get the error
				plugin, err := repo.Get(id)
				if err != nil {
					log.Error(ctx, "Error getting updated plugin after enable failure", "id", id, err)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				// Return error response with message field for React-Admin compatibility
				// and include the plugin data so UI can update its state
				errorResponse := struct {
					Message string        `json:"message"`
					Plugin  *model.Plugin `json:"plugin"`
				}{
					Message: enableErr.Error(),
					Plugin:  plugin,
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				_ = json.NewEncoder(w).Encode(errorResponse)
				return
			}
		} else {
			if err := api.pluginManager.DisablePlugin(ctx, id); err != nil {
				log.Error(ctx, "Error disabling plugin", "id", id, err)
				http.Error(w, "Error disabling plugin: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// Refresh and return updated plugin
	plugin, err := repo.Get(id)
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

// validateAndUpdateConfig validates the config JSON and updates the plugin.
// Returns an error if validation or update fails (error response already written).
func validateAndUpdateConfig(ctx context.Context, pm PluginManager, id, configJSON string, w http.ResponseWriter) error {
	if configJSON != "" && !isValidJSON(configJSON) {
		http.Error(w, "Invalid JSON in config field", http.StatusBadRequest)
		return errors.New("invalid JSON")
	}
	if err := pm.UpdatePluginConfig(ctx, id, configJSON); err != nil {
		log.Error(ctx, "Error updating plugin config", "id", id, err)
		http.Error(w, "Error updating plugin configuration: "+err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

// validateAndUpdateUsers validates the users JSON and updates the plugin.
// Returns an error if validation or update fails (error response already written).
func validateAndUpdateUsers(ctx context.Context, pm PluginManager, repo model.PluginRepository, id string, req PluginUpdateRequest, w http.ResponseWriter) error {
	// Get current values if not provided in request
	plugin, err := repo.Get(id)
	if err != nil {
		log.Error(ctx, "Error getting plugin for users update", "id", id, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return err
	}

	usersJSON := plugin.Users
	allUsers := plugin.AllUsers

	if req.Users != nil {
		if *req.Users != "" && !isValidJSON(*req.Users) {
			http.Error(w, "Invalid JSON in users field", http.StatusBadRequest)
			return errors.New("invalid JSON")
		}
		usersJSON = *req.Users
	}
	if req.AllUsers != nil {
		allUsers = *req.AllUsers
	}

	if err := pm.UpdatePluginUsers(ctx, id, usersJSON, allUsers); err != nil {
		log.Error(ctx, "Error updating plugin users", "id", id, err)
		http.Error(w, "Error updating plugin users: "+err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

// validateAndUpdateLibraries validates the libraries JSON and updates the plugin.
// Returns an error if validation or update fails (error response already written).
func validateAndUpdateLibraries(ctx context.Context, pm PluginManager, repo model.PluginRepository, id string, req PluginUpdateRequest, w http.ResponseWriter) error {
	// Get current values if not provided in request
	plugin, err := repo.Get(id)
	if err != nil {
		log.Error(ctx, "Error getting plugin for libraries update", "id", id, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return err
	}

	librariesJSON := plugin.Libraries
	allLibraries := plugin.AllLibraries

	if req.Libraries != nil {
		if *req.Libraries != "" && !isValidJSON(*req.Libraries) {
			http.Error(w, "Invalid JSON in libraries field", http.StatusBadRequest)
			return errors.New("invalid JSON")
		}
		librariesJSON = *req.Libraries
	}
	if req.AllLibraries != nil {
		allLibraries = *req.AllLibraries
	}

	if err := pm.UpdatePluginLibraries(ctx, id, librariesJSON, allLibraries); err != nil {
		log.Error(ctx, "Error updating plugin libraries", "id", id, err)
		http.Error(w, "Error updating plugin libraries: "+err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}
