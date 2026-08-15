package nativeapi

import (
	"errors"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func deleteMediaFile(maintenance core.Maintenance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing media file id", http.StatusBadRequest)
			return
		}

		err := maintenance.DeleteMediaFile(r.Context(), id)
		switch {
		case err == nil:
			writeDeleteManyResponse(w, r, []string{id})
		case errors.Is(err, model.ErrNotAuthorized), errors.Is(err, core.ErrMediaFileDeletionDisabled):
			http.Error(w, err.Error(), http.StatusForbidden)
		case errors.Is(err, model.ErrNotFound), errors.Is(err, fs.ErrNotExist):
			http.Error(w, "not found", http.StatusNotFound)
		case errors.Is(err, core.ErrMediaFileDeletionUnsupported):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			log.Error(r.Context(), "Error deleting media file", "id", id, err)
			http.Error(w, "failed to delete media file", http.StatusInternalServerError)
		}
	}
}
