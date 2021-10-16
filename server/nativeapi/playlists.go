package nativeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

type restHandler = func(rest.RepositoryConstructor, ...rest.Logger) http.HandlerFunc

func getPlaylist(ds model.DataStore) http.HandlerFunc {
	// Add a middleware to capture the playlistId
	wrapper := func(handler restHandler) http.HandlerFunc {
		return func(res http.ResponseWriter, req *http.Request) {
			constructor := func(ctx context.Context) rest.Repository {
				plsRepo := ds.Playlist(ctx)
				plsId := chi.URLParam(req, "playlistId")
				return plsRepo.Tracks(plsId)
			}

			handler(constructor).ServeHTTP(res, req)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("accept")
		if strings.ToLower(accept) == "audio/x-mpegurl" {
			handleExportPlaylist(ds)(w, r)
			return
		}
		wrapper(rest.GetAll)(w, r)
	}
}

func handleExportPlaylist(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		plsRepo := ds.Playlist(ctx)
		plsId := chi.URLParam(r, "playlistId")
		pls, err := plsRepo.GetWithTracks(plsId)
		if err == model.ErrNotFound {
			log.Warn("Playlist not found", "playlistId", plsId)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Error("Error retrieving the playlist", "playlistId", plsId, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Debug(ctx, "Exporting playlist as M3U", "playlistId", plsId, "name", pls.Name)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		disposition := fmt.Sprintf("attachment; filename=\"%s.m3u\"", pls.Name)
		w.Header().Set("Content-Disposition", disposition)

		// TODO: Move this and the import playlist logic to `core`
		_, err = w.Write([]byte("#EXTM3U\n"))
		if err != nil {
			log.Error(ctx, "Error sending playlist", "name", pls.Name)
			return
		}
		for _, t := range pls.Tracks {
			header := fmt.Sprintf("#EXTINF:%.f,%s - %s\n", t.Duration, t.Artist, t.Title)
			line := t.Path + "\n"
			_, err = w.Write([]byte(header + line))
			if err != nil {
				log.Error(ctx, "Error sending playlist", "name", pls.Name)
				return
			}
		}
	}
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
	type addTracksPayload struct {
		Ids       []string       `json:"ids"`
		AlbumIds  []string       `json:"albumIds"`
		ArtistIds []string       `json:"artistIds"`
		Discs     []model.DiscID `json:"discs"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		playlistId := utils.ParamString(r, ":playlistId")
		var payload addTracksPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		tracksRepo := ds.Playlist(r.Context()).Tracks(playlistId)
		count, c := 0, 0
		if c, err = tracksRepo.Add(payload.Ids); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		count += c
		if c, err = tracksRepo.AddAlbums(payload.AlbumIds); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		count += c
		if c, err = tracksRepo.AddArtists(payload.ArtistIds); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		count += c
		if c, err = tracksRepo.AddDiscs(payload.Discs); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		count += c

		// Must return an object with an ID, to satisfy ReactAdmin `create` call
		_, err = fmt.Fprintf(w, `{"added":%d}`, count)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func reorderItem(ds model.DataStore) http.HandlerFunc {
	type reorderPayload struct {
		InsertBefore string `json:"insert_before"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		playlistId := utils.ParamString(r, ":playlistId")
		id := utils.ParamInt(r, ":id", 0)
		if id == 0 {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		var payload reorderPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		newPos, err := strconv.Atoi(payload.InsertBefore)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		tracksRepo := ds.Playlist(r.Context()).Tracks(playlistId)
		err = tracksRepo.Reorder(id, newPos)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err = w.Write([]byte(fmt.Sprintf(`{"id":"%d"}`, id)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
