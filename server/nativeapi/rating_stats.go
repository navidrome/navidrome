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
		stats, err := ds.User(r.Context()).RatingStats()
		if err != nil {
			log.Error(r.Context(), "Error getting rating stats", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !currentUser.IsAdmin {
			filtered := make([]model.UserRatingStats, 0, 1)
			for _, s := range stats {
				if s.UserID == currentUser.ID {
					filtered = append(filtered, s)
					break
				}
			}
			stats = filtered
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
		if err != nil || rating < 1 || rating > 5 || userID == "" || (itemType != "album" && itemType != "song") {
			http.Error(w, "invalid parameters", http.StatusBadRequest)
			return
		}

		if !currentUser.IsAdmin && currentUser.ID != userID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		items, err := ds.User(r.Context()).RatingItems(userID, itemType, rating)
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
