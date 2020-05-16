package app

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/utils"
)

type addTracksPayload struct {
	Ids []string `json:"ids"`
}

func deleteFromPlaylist(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playlistId := utils.ParamString(r, ":playlistId")
		id := r.URL.Query().Get(":id")
		tracksRepo := ds.Playlist(r.Context()).Tracks(playlistId)
		err := tracksRepo.Delete(id)
		if err == model.ErrNotFound {
			log.Warn("Track not found in playlist", "playlistId", playlistId, "id", id)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Error("Error deleting track from playlist", "playlistId", playlistId, "id", id, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = w.Write([]byte("{}"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func addToPlaylist(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playlistId := utils.ParamString(r, ":playlistId")
		tracksRepo := ds.Playlist(r.Context()).Tracks(playlistId)
		var payload addTracksPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = tracksRepo.Add(payload.Ids)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Must return an object with an ID, to satisfy ReactAdmin `create` call
		_, err = w.Write([]byte(fmt.Sprintf(`{"id":"%s"}`, playlistId)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
