package nativeapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

func (api *Router) addRatingStatsRoute(r chi.Router) {
	r.Get("/ratingStats", getRatingStats(api.ds))
	r.Get("/ratingItems", getRatingItems(api.ds))
}

func getRatingStats(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := request.UserFrom(r.Context())
		filterUserID := ""
		if !currentUser.IsAdmin {
			filterUserID = currentUser.ID
		}
		stats, err := ds.User(r.Context()).RatingStats(r.Context(), filterUserID)
		if err != nil {
			log.Error(r.Context(), "Error getting rating stats", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			log.Error(r.Context(), "Error encoding rating stats", err)
		}
	}
}

func getRatingItems(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := request.UserFrom(r.Context())
		userID := r.URL.Query().Get("userId")
		itemType := r.URL.Query().Get("type")
		ratingStr := r.URL.Query().Get("rating")

		rating, err := strconv.Atoi(ratingStr)
		if err != nil || rating < 1 || rating > 5 {
			http.Error(w, "rating must be between 1 and 5", http.StatusBadRequest)
			return
		}
		if userID == "" {
			http.Error(w, "userId is required", http.StatusBadRequest)
			return
		}
		if itemType != "album" && itemType != "song" {
			http.Error(w, "type must be 'album' or 'song'", http.StatusBadRequest)
			return
		}

		if !currentUser.IsAdmin && currentUser.ID != userID {
			http.Error(w, "non-admin users can only query their own ratings", http.StatusForbidden)
			return
		}

		items, err := ds.User(r.Context()).RatingItems(r.Context(), userID, itemType, rating)
		if err != nil {
			log.Error(r.Context(), "Error getting rating items", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(items); err != nil {
			log.Error(r.Context(), "Error encoding rating items", err)
		}
	}
}
