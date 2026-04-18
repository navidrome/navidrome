package nativeapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/tageditor"
)

func (api *Router) addTagEditorRoute(r chi.Router) {
	service := tageditor.New(api.ds, api.scanner)

	r.Route("/tag-editor", func(r chi.Router) {
		r.Route("/song/{id}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				payload, err := service.GetSong(r.Context(), chi.URLParam(r, "id"))
				if err != nil {
					writeTagEditorError(w, err, http.StatusBadRequest)
					return
				}
				writeTagEditorJSON(w, http.StatusOK, payload)
			})
			r.Put("/", func(w http.ResponseWriter, r *http.Request) {
				var payload tageditor.SongPayload
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					writeTagEditorError(w, err, http.StatusBadRequest)
					return
				}
				updated, err := service.UpdateSong(r.Context(), chi.URLParam(r, "id"), payload)
				if err != nil {
					writeTagEditorError(w, err, http.StatusBadRequest)
					return
				}
				writeTagEditorJSON(w, http.StatusOK, updated)
			})
		})

		r.Route("/album/{id}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				payload, err := service.GetAlbum(r.Context(), chi.URLParam(r, "id"))
				if err != nil {
					writeTagEditorError(w, err, http.StatusBadRequest)
					return
				}
				writeTagEditorJSON(w, http.StatusOK, payload)
			})
			r.Put("/", func(w http.ResponseWriter, r *http.Request) {
				var payload tageditor.AlbumPayload
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					writeTagEditorError(w, err, http.StatusBadRequest)
					return
				}
				updated, err := service.UpdateAlbum(r.Context(), chi.URLParam(r, "id"), payload)
				if err != nil {
					writeTagEditorError(w, err, http.StatusBadRequest)
					return
				}
				writeTagEditorJSON(w, http.StatusOK, updated)
			})
		})
	})
}

func writeTagEditorJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeTagEditorError(w http.ResponseWriter, err error, status int) {
	writeTagEditorJSON(w, status, map[string]string{"error": err.Error()})
}
