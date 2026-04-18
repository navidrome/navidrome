package nativeapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/request"
)

// RecommendationItem represents a single recommendation returned to the UI
type RecommendationItem struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Album      string  `json:"album"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason,omitempty"`
	CoverArtID string  `json:"coverArtId,omitempty"`
}

// RecommendationResponse is the full response from the serving container
type RecommendationResponse struct {
	UserID          string               `json:"userId"`
	Recommendations []RecommendationItem `json:"recommendations"`
	ModelVersion    string               `json:"modelVersion,omitempty"`
	GeneratedAt     string               `json:"generatedAt,omitempty"`
}

func (api *Router) addRecommendationRoute(r chi.Router) {
	r.Route("/recommendation", func(r chi.Router) {
		r.Get("/", api.getRecommendations())
	})
}

func (api *Router) getRecommendations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)

		if !conf.Server.EnableRecommendations {
			http.Error(w, "Recommendations are disabled", http.StatusNotFound)
			return
		}

		serviceURL := conf.Server.RecommendationServiceURL
		if serviceURL == "" {
			log.Warn(ctx, "Recommendation service URL not configured")
			// Return empty recommendations instead of error
			respondWithEmptyRecommendations(w, user.UserName)
			return
		}

		// Call the serving container
		reqURL := fmt.Sprintf("%s/recommend?user_id=%s&top_n=20", serviceURL, user.UserName)
		client := &http.Client{Timeout: 5 * time.Second}

		resp, err := client.Get(reqURL)
		if err != nil {
			log.Error(ctx, "Failed to call recommendation service", "url", reqURL, err)
			// Graceful fallback: return empty recommendations
			respondWithEmptyRecommendations(w, user.UserName)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Warn(ctx, "Recommendation service returned error", "status", resp.StatusCode, "body", string(body))
			respondWithEmptyRecommendations(w, user.UserName)
			return
		}

		// Forward the response from the serving container
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error(ctx, "Failed to read recommendation response", err)
			respondWithEmptyRecommendations(w, user.UserName)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(body) //nolint:errcheck
	}
}

func respondWithEmptyRecommendations(w http.ResponseWriter, userID string) {
	resp := RecommendationResponse{
		UserID:          userID,
		Recommendations: []RecommendationItem{},
		ModelVersion:    "none",
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}
