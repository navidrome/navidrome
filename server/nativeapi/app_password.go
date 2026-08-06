package nativeapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// addAppPasswordRoute mounts the /user/{userId}/app-password CRUD endpoints.
//
// The route is owner-or-admin gated: a non-admin user can only manage their
// own passwords; an admin can manage any user's. POST returns the freshly
// generated plaintext secret exactly once; subsequent GETs only return
// metadata.
func (api *Router) addAppPasswordRoute(r chi.Router) {
	r.Route("/user/{userId}/app-password", func(r chi.Router) {
		r.Use(appPasswordOwnerOrAdminMiddleware)
		r.Get("/", listAppPasswords(api.ds))
		r.Post("/", createAppPassword(api.ds))
		r.Delete("/{id}", deleteAppPassword(api.ds))
	})
}

// appPasswordOwnerOrAdminMiddleware permits the request only if the
// authenticated user owns the userId in the path or is an admin. Any other
// caller (including unauthenticated requests, which should not reach this
// far in normal routing) gets 403.
func appPasswordOwnerOrAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller, ok := request.UserFrom(r.Context())
		if !ok {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		userID := chi.URLParam(r, "userId")
		if !caller.IsAdmin && caller.ID != userID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// listAppPasswords returns the metadata-only public projection of the
// caller's (or, for admins, the target user's) app passwords. The plaintext
// secret is never re-served — it is only returned by POST at creation time.
func listAppPasswords(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userId")
		rows, err := ds.AppPassword(r.Context()).ListForUser(r.Context(), userID)
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, rows)
	}
}

// createAppPasswordRequest is the JSON payload accepted by POST.
type createAppPasswordRequest struct {
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// createAppPasswordResponse is the JSON payload returned by POST. The
// "secret" field is the plaintext shown only once.
type createAppPasswordResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Secret    string     `json:"secret"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// createAppPassword generates and stores a new app password and returns the
// plaintext secret in the response body. Callers must surface the secret to
// the user immediately — Navidrome cannot retrieve it again.
func createAppPassword(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userId")

		var req createAppPasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			_ = rest.RespondWithError(w, http.StatusUnprocessableEntity, "invalid request body")
			return
		}
		if req.Name == "" {
			_ = rest.RespondWithError(w, http.StatusUnprocessableEntity, "name is required")
			return
		}

		plaintext, ap, err := ds.AppPassword(r.Context()).Create(r.Context(), userID, req.Name, req.ExpiresAt)
		if err != nil {
			log.Error(r, "Failed to create app password", "userId", userID, err)
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusCreated, createAppPasswordResponse{
			ID:        ap.ID,
			Name:      ap.Name,
			Secret:    plaintext,
			CreatedAt: ap.CreatedAt,
			ExpiresAt: ap.ExpiresAt,
		})
	}
}

// deleteAppPassword removes an app password the caller is allowed to
// manage. The repository enforces the ownership check redundantly: even a
// path-confusion attack that smuggled a foreign userId past the middleware
// would not delete other users' rows because the WHERE clause includes
// user_id.
func deleteAppPassword(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userId")
		appPwdID := chi.URLParam(r, "id")
		if err := ds.AppPassword(r.Context()).Delete(r.Context(), appPwdID, userID); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				_ = rest.RespondWithError(w, http.StatusNotFound, "not found")
				return
			}
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
