package app

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/utils"
)

type addTracksPayload struct {
	Ids []string `json:"ids"`
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
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
}
