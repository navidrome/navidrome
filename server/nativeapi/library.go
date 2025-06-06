package nativeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Masterminds/squirrel"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// Library management endpoints (admin only)
func (n *Router) addLibraryRoute(r chi.Router) {
	r.Route("/library", func(r chi.Router) {
		r.Use(adminOnlyMiddleware)
		r.Get("/", getLibraries(n.ds))
		r.Post("/", createLibrary(n.ds))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(parseIDMiddleware)
			r.Get("/", getLibrary(n.ds))
			r.Put("/", updateLibrary(n.ds))
			r.Delete("/", deleteLibrary(n.ds))
		})
	})
}

// User-library association endpoints (admin only)
func (n *Router) addUserLibraryRoute(r chi.Router) {
	r.Route("/user/{id}/library", func(r chi.Router) {
		r.Use(adminOnlyMiddleware)
		r.Use(parseUserIDMiddleware)
		r.Get("/", getUserLibraries(n.ds))
		r.Put("/", setUserLibraries(n.ds))
	})
}

// Middleware to ensure only admin users can access endpoints
func adminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := request.UserFrom(r.Context())
		if !ok || !user.IsAdmin {
			http.Error(w, "Access denied: admin privileges required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
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

func getLibraries(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		libraries, err := ds.Library(r.Context()).GetAll()
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

func getLibrary(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		libraryID := r.Context().Value("libraryID").(int)

		library, err := ds.Library(r.Context()).Get(libraryID)
		if err != nil {
			if err == model.ErrNotFound {
				http.Error(w, "Library not found", http.StatusNotFound)
				return
			}
			log.Error(r.Context(), "Error getting library", "libraryID", libraryID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(library); err != nil {
			log.Error(r.Context(), "Error encoding library response", err)
		}
	}
}

func createLibrary(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var library model.Library
		if err := json.NewDecoder(r.Body).Decode(&library); err != nil {
			log.Error(r.Context(), "Error decoding library request", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if library.Name == "" {
			http.Error(w, "Library name is required", http.StatusBadRequest)
			return
		}
		if library.Path == "" {
			http.Error(w, "Library path is required", http.StatusBadRequest)
			return
		}

		if err := ds.Library(r.Context()).Put(&library); err != nil {
			log.Error(r.Context(), "Error creating library", err)
			http.Error(w, "Failed to create library", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(library); err != nil {
			log.Error(r.Context(), "Error encoding library response", err)
		}
	}
}

func updateLibrary(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		libraryID := r.Context().Value("libraryID").(int)

		var library model.Library
		if err := json.NewDecoder(r.Body).Decode(&library); err != nil {
			log.Error(r.Context(), "Error decoding library request", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Ensure the ID matches the URL parameter
		library.ID = libraryID

		// Validate required fields
		if library.Name == "" {
			http.Error(w, "Library name is required", http.StatusBadRequest)
			return
		}
		if library.Path == "" {
			http.Error(w, "Library path is required", http.StatusBadRequest)
			return
		}

		if err := ds.Library(r.Context()).Put(&library); err != nil {
			if err == model.ErrNotFound {
				http.Error(w, "Library not found", http.StatusNotFound)
				return
			}
			log.Error(r.Context(), "Error updating library", "libraryID", libraryID, err)
			http.Error(w, "Failed to update library", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(library); err != nil {
			log.Error(r.Context(), "Error encoding library response", err)
		}
	}
}

func deleteLibrary(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		libraryID := r.Context().Value("libraryID").(int)

		if err := ds.Library(r.Context()).Delete(libraryID); err != nil {
			if err == model.ErrNotFound {
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

func getUserLibraries(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		// Verify user exists
		if _, err := ds.User(r.Context()).Get(userID); err != nil {
			if err == model.ErrNotFound {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			log.Error(r.Context(), "Error getting user", "userID", userID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		libraries, err := ds.User(r.Context()).GetUserLibraries(userID)
		if err != nil {
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

func setUserLibraries(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		// Verify user exists
		user, err := ds.User(r.Context()).Get(userID)
		if err != nil {
			if err == model.ErrNotFound {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			log.Error(r.Context(), "Error getting user", "userID", userID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		var request struct {
			LibraryIDs []int `json:"libraryIds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			log.Error(r.Context(), "Error decoding request", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Admin users get all libraries automatically - don't allow manual assignment
		if user.IsAdmin {
			http.Error(w, "Cannot manually assign libraries to admin users", http.StatusBadRequest)
			return
		}

		// Validate library IDs exist
		if len(request.LibraryIDs) > 0 {
			// Use CountAll with IN filter to efficiently check if all library IDs exist
			count, err := ds.Library(r.Context()).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"id": request.LibraryIDs},
			})
			if err != nil {
				log.Error(r.Context(), "Error counting libraries", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if int(count) != len(request.LibraryIDs) {
				// Some library IDs don't exist - find which ones for better error message
				existingLibraries, err := ds.Library(r.Context()).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"id": request.LibraryIDs},
				})
				if err != nil {
					log.Error(r.Context(), "Error getting all libraries", err)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}

				libraryMap := make(map[int]bool)
				for _, lib := range existingLibraries {
					libraryMap[lib.ID] = true
				}

				for _, libID := range request.LibraryIDs {
					if !libraryMap[libID] {
						http.Error(w, fmt.Sprintf("Library ID %d does not exist", libID), http.StatusBadRequest)
						return
					}
				}
			}
		}

		// Regular users must have at least one library
		if len(request.LibraryIDs) == 0 {
			http.Error(w, "At least one library must be assigned to non-admin users", http.StatusBadRequest)
			return
		}

		if err := ds.User(r.Context()).SetUserLibraries(userID, request.LibraryIDs); err != nil {
			log.Error(r.Context(), "Error setting user libraries", "userID", userID, err)
			http.Error(w, "Failed to set user libraries", http.StatusInternalServerError)
			return
		}

		// Return updated user libraries
		libraries, err := ds.User(r.Context()).GetUserLibraries(userID)
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
