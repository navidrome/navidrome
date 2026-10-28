package apiv1

import (
	"net/http"
	"strconv"

	"github.com/SunilSherkhane/navidrome/core"
	"github.com/go-chi/chi/v5"
)

func (s *httpHandler) handleTopArtists(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserID(r)
	timeRange := r.URL.Query().Get("timeRange")
	if timeRange == "" {
		timeRange = "30d"
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 20
	}
	items, err := s.statsService.GetTopArtists(r.Context(), userID, timeRange, limit)
	if err != nil {
		s.respondError(w, r, err)
		return
	}
	s.respondJSON(w, r, http.StatusOK, items)
}

func (s *httpHandler) handleTopAlbums(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserID(r)
	timeRange := r.URL.Query().Get("timeRange")
	if timeRange == "" {
		timeRange = "30d"
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 20
	}
	items, err := s.statsService.GetTopAlbums(r.Context(), userID, timeRange, limit)
	if err != nil {
		s.respondError(w, r, err)
		return
	}
	s.respondJSON(w, r, http.StatusOK, items)
}

func (s *httpHandler) handleTopSongs(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserID(r)
	timeRange := r.URL.Query().Get("timeRange")
	if timeRange == "" {
		timeRange = "30d"
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 20
	}
	items, err := s.statsService.GetTopSongs(r.Context(), userID, timeRange, limit)
	if err != nil {
		s.respondError(w, r, err)
		return
	}
	s.respondJSON(w, r, http.StatusOK, items)
}
