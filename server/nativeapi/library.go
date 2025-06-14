package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// User-library association endpoints (admin only)
func (n *Router) addUserLibraryRoute(r chi.Router) {
	r.Route("/user/{id}/library", func(r chi.Router) {
		r.Use(parseUserIDMiddleware)
		r.Get("/", getUserLibraries(n.libs))
		r.Put("/", setUserLibraries(n.libs))
	})
}

// Middleware to parse user ID from URL
func parseUserIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "id")
		if userID == "" {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// User-library association handlers

func getUserLibraries(service core.Library) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		libraries, err := service.GetUserLibraries(r.Context(), userID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			log.Error(r.Context(), "Error getting user libraries", "userID", userID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(libraries); err != nil {
			log.Error(r.Context(), "Error encoding user libraries response", err)
		}
	}
}

func setUserLibraries(service core.Library) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		var request struct {
			LibraryIDs []int `json:"libraryIds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			log.Error(r.Context(), "Error decoding request", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := service.SetUserLibraries(r.Context(), userID, request.LibraryIDs); err != nil {
			log.Error(r.Context(), "Error setting user libraries", "userID", userID, err)
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			if errors.Is(err, model.ErrValidation) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, "Failed to set user libraries", http.StatusInternalServerError)
			return
		}

		// Return updated user libraries
		libraries, err := service.GetUserLibraries(r.Context(), userID)
		if err != nil {
			log.Error(r.Context(), "Error getting updated user libraries", "userID", userID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(libraries); err != nil {
			log.Error(r.Context(), "Error encoding user libraries response", err)
		}
	}
}
