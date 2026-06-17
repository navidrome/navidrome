package metadatamanager

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service MetadataService
}

func NewHandler(s MetadataService) *Handler {
	return &Handler{service: s}
}

func (h *Handler) BindRoutes(r chi.Router) {
	r.Post("/song/{id}/tag", h.UpdateSong)
	r.Post("/song/{id}/artwork", h.UpdateArtwork)
}

func (h *Handler) UpdateSong(w http.ResponseWriter, r *http.Request) {
	songID := chi.URLParam(r, "id")
	if songID == "" {
		http.Error(w, "Missing song identifier", http.StatusBadRequest)
		return
	}

	var tags map[string]string
	if err := json.NewDecoder(r.Body).Decode(&tags); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateTags(r.Context(), songID, tags); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateArtwork(w http.ResponseWriter, r *http.Request) {
	songID := chi.URLParam(r, "id")
	if songID == "" {
		http.Error(w, "Missing song identifier", http.StatusBadRequest)
		return
	}

	mimeType := r.Header.Get("Content-Type")
	if err := h.service.UpdateArtwork(r.Context(), songID, r.Body, mimeType); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	w.WriteHeader(http.StatusNoContent)
}
