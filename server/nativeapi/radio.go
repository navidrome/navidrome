package nativeapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/agents/radiobrowser"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func submitClick(ds model.DataStore) http.HandlerFunc {
	browser := radiobrowser.GetRadioBrowser(ds)

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		radioId := chi.URLParam(r, "radioId")
		err := browser.SubmitClick(ctx, radioId)

		if err != nil {
			log.Error(r.Context(), "Error submitting click", "radio id", radioId, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte(``))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
