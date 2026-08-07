package nativeapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// User-feature-permission association endpoints (admin only)
func (api *Router) addUserFeaturePermissionsRoute(r chi.Router) {
	r.Route("/user/{id}/featurePermissions", func(r chi.Router) {
		r.Use(parseUserIDMiddleware)
		r.Get("/", getUserFeaturePermissions(api.users))
		r.Put("/", setUserFeaturePermissions(api.users))
	})
}

func getUserFeaturePermissions(service core.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		permissions, err := service.GetUserFeaturePermissions(r.Context(), userID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			log.Error(r.Context(), "Error getting user feature permissions", "userID", userID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(permissions); err != nil {
			log.Error(r.Context(), "Error encoding user feature permissions response", err)
		}
	}
}

func setUserFeaturePermissions(service core.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		var request struct {
			Permissions map[string]bool `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			log.Error(r.Context(), "Error decoding request", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		permissions, err := service.SetUserFeaturePermissions(r.Context(), userID, request.Permissions)
		if err != nil {
			log.Error(r.Context(), "Error setting user feature permissions", "userID", userID, err)
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			if errors.Is(err, model.ErrValidation) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, "Failed to set user feature permissions", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(permissions); err != nil {
			log.Error(r.Context(), "Error encoding user feature permissions response", err)
		}
	}
}
