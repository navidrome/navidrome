package nativeapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func (api *Router) addRatingStatsRoute(r chi.Router) {
	r.Get("/ratingStats", getRatingStats(api.ds))
	r.Get("/ratingItems", getRatingItems(api.ds))
}

func getRatingStats(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := ds.User(r.Context()).RatingStats()
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
		userID := r.URL.Query().Get("userId")
		itemType := r.URL.Query().Get("type")
		ratingStr := r.URL.Query().Get("rating")

		rating, err := strconv.Atoi(ratingStr)
		if err != nil || userID == "" || (itemType != "album" && itemType != "song") {
			http.Error(w, "invalid parameters", http.StatusBadRequest)
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
