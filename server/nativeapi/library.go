package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// Library management endpoints (admin only)
func (n *Router) addLibraryRoute(r chi.Router) {
	r.Route("/library", func(r chi.Router) {
		r.Get("/", getLibraries(n.libs))
		r.Post("/", createLibrary(n.libs))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(parseIDMiddleware)
			r.Get("/", getLibrary(n.libs))
			r.Put("/", updateLibrary(n.libs))
			r.Delete("/", deleteLibrary(n.libs))
		})
	})
}

// User-library association endpoints (admin only)
func (n *Router) addUserLibraryRoute(r chi.Router) {
	r.Route("/user/{id}/library", func(r chi.Router) {
		r.Use(parseUserIDMiddleware)
		r.Get("/", getUserLibraries(n.libs))
		r.Put("/", setUserLibraries(n.libs))
	})
}

// Middleware to parse library ID from URL
func parseIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid library ID", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), "libraryID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
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

// Library CRUD handlers

func getLibraries(service core.Library) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		libraries, err := service.GetAll(r.Context())
		if err != nil {
			log.Error(r.Context(), "Error getting libraries", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(libraries); err != nil {
			log.Error(r.Context(), "Error encoding libraries response", err)
		}
	}
}

func getLibrary(service core.Library) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		libraryID := r.Context().Value("libraryID").(int)

		lib, err := service.Get(r.Context(), libraryID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, "Library not found", http.StatusNotFound)
				return
			}
			log.Error(r.Context(), "Error getting library", "libraryID", libraryID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lib); err != nil {
			log.Error(r.Context(), "Error encoding library response", err)
		}
	}
}

func createLibrary(service core.Library) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var lib model.Library
		if err := json.NewDecoder(r.Body).Decode(&lib); err != nil {
			log.Error(r.Context(), "Error decoding library request", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := service.Create(r.Context(), &lib); err != nil {
			log.Error(r.Context(), "Error creating library", err)
			if errors.Is(err, model.ErrValidation) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, "Failed to create library", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(lib); err != nil {
			log.Error(r.Context(), "Error encoding library response", err)
		}
	}
}

func updateLibrary(service core.Library) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		libraryID := r.Context().Value("libraryID").(int)

		var lib model.Library
		if err := json.NewDecoder(r.Body).Decode(&lib); err != nil {
			log.Error(r.Context(), "Error decoding library request", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Ensure the ID matches the URL parameter
		lib.ID = libraryID

		if err := service.Update(r.Context(), &lib); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, "Library not found", http.StatusNotFound)
				return
			}
			if errors.Is(err, model.ErrValidation) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			log.Error(r.Context(), "Error updating library", "libraryID", libraryID, err)
			http.Error(w, "Failed to update library", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lib); err != nil {
			log.Error(r.Context(), "Error encoding library response", err)
		}
	}
}

func deleteLibrary(service core.Library) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		libraryID := r.Context().Value("libraryID").(int)

		if err := service.Delete(r.Context(), libraryID); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, "Library not found", http.StatusNotFound)
				return
			}
			log.Error(r.Context(), "Error deleting library", "libraryID", libraryID, err)
			http.Error(w, "Failed to delete library", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
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
