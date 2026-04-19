package nativeapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// RecommendationItem represents a single recommendation returned to the UI
type RecommendationItem struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Artist  string  `json:"artist"`
	Album   string  `json:"album"`
	Score   float64 `json:"score"`
	Rank    int     `json:"rank,omitempty"`
	TrackID string  `json:"track_id,omitempty"`
}

// RecommendationResponse is the response sent to the UI
type RecommendationResponse struct {
	UserID          string               `json:"userId"`
	Recommendations []RecommendationItem `json:"recommendations"`
	ModelVersion    string               `json:"modelVersion,omitempty"`
	GeneratedAt     string               `json:"generatedAt,omitempty"`
}

// serveRecommendRequest matches the serving container's /recommend-by-tracks schema
type serveRecommendRequest struct {
	SessionID       string   `json:"session_id"`
	UserID          string   `json:"user_id"`
	TrackIDs        []string `json:"track_ids"`
	ExcludeTrackIDs []string `json:"exclude_track_ids"`
	TopN            int      `json:"top_n"`
}

// serveRecommendResponse matches the serving container's response
type serveRecommendResponse struct {
	Recommendations []struct {
		Rank    int     `json:"rank"`
		ItemIdx int     `json:"item_idx"`
		TrackID string  `json:"track_id"`
		Score   float64 `json:"score"`
	} `json:"recommendations"`
	ModelVersion       string  `json:"model_version"`
	GeneratedAt        string  `json:"generated_at"`
	InferenceLatencyMs float64 `json:"inference_latency_ms"`
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
			respondWithEmptyRecommendations(w, user.UserName)
			return
		}

		// Get some tracks from the library to use as seed for recommendations.
		// Uses random tracks — in production this would use the user's play history.
		mfRepo := api.ds.MediaFile(ctx)
		songs, err := mfRepo.GetAll(model.QueryOptions{Max: 10, Sort: "random"})

		var trackIDs []string
		if err == nil {
			for _, mf := range songs {
				trackIDs = append(trackIDs, mf.ID)
			}
		}

		if len(trackIDs) == 0 {
			respondWithEmptyRecommendations(w, user.UserName)
			return
		}

		// Call serving container: POST /recommend-by-tracks
		reqBody := serveRecommendRequest{
			SessionID: "navidrome-ui-" + user.UserName,
			UserID:    user.UserName,
			TrackIDs:  trackIDs,
			TopN:      20,
		}
		jsonBody, _ := json.Marshal(reqBody)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Post(
			serviceURL+"/recommend-by-tracks",
			"application/json",
			bytes.NewReader(jsonBody),
		)
		if err != nil {
			log.Error(ctx, "Failed to call recommendation service", err)
			respondWithEmptyRecommendations(w, user.UserName)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			log.Warn(ctx, "Recommendation service error", "status", resp.StatusCode, "body", string(body))
			respondWithEmptyRecommendations(w, user.UserName)
			return
		}

		// Parse serving response and convert to UI format
		var serveResp serveRecommendResponse
		if err := json.Unmarshal(body, &serveResp); err != nil {
			log.Error(ctx, "Failed to parse recommendation response", err)
			respondWithEmptyRecommendations(w, user.UserName)
			return
		}

		var recs []RecommendationItem
		for _, rec := range serveResp.Recommendations {
			recs = append(recs, RecommendationItem{
				ID:      rec.TrackID,
				TrackID: rec.TrackID,
				Score:   rec.Score,
				Rank:    rec.Rank,
				Title:   "Track " + rec.TrackID,
				Artist:  "Recommended",
			})
		}

		uiResp := RecommendationResponse{
			UserID:          user.UserName,
			Recommendations: recs,
			ModelVersion:    serveResp.ModelVersion,
			GeneratedAt:     serveResp.GeneratedAt,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(uiResp) //nolint:errcheck
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
