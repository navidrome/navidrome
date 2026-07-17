package nativeapi

import (
	"encoding/json"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
)

func (api *Router) addMediaFileTagRoutes(r chi.Router) {
	r.Route("/mediaFileTag", func(r chi.Router) {
		r.Get("/", api.tagsForSong())
		r.Get("/names", api.allTagNames())
		r.Post("/", api.tagSong())
		r.Delete("/", api.untagSong())
	})
}

type mediaFileTagPayload struct {
	MediaFileID string `json:"mediaFileId"`
	TagName     string `json:"tagName"`
}

func (api *Router) tagsForSong() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mediaFileID := r.URL.Query().Get("media_file_id")
		if mediaFileID == "" {
			http.Error(w, "media_file_id is required", http.StatusBadRequest)
			return
		}
		tags, err := api.ds.MediaFileTag(r.Context()).TagsForSong(mediaFileID)
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, tags)
	}
}

func (api *Router) allTagNames() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tags, err := api.ds.MediaFileTag(r.Context()).AllTagNames()
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, tags)
	}
}

func (api *Router) tagSong() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p mediaFileTagPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if p.MediaFileID == "" || p.TagName == "" {
			http.Error(w, "mediaFileId and tagName are required", http.StatusBadRequest)
			return
		}
		if err := api.ds.MediaFileTag(r.Context()).TagSong(p.MediaFileID, p.TagName); err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, p)
	}
}

func (api *Router) untagSong() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p mediaFileTagPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if p.MediaFileID == "" || p.TagName == "" {
			http.Error(w, "mediaFileId and tagName are required", http.StatusBadRequest)
			return
		}
		if err := api.ds.MediaFileTag(r.Context()).UntagSong(p.MediaFileID, p.TagName); err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, p)
	}
}
