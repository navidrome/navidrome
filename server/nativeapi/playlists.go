package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/req"
)

type restHandler = func(rest.RepositoryConstructor, ...rest.Logger) http.HandlerFunc

func getPlaylist(ds model.DataStore) http.HandlerFunc {
	// Add a middleware to capture the playlistId
	wrapper := func(handler restHandler) http.HandlerFunc {
		return func(res http.ResponseWriter, req *http.Request) {
			constructor := func(ctx context.Context) rest.Repository {
				plsRepo := ds.Playlist(ctx)
				plsId := chi.URLParam(req, "playlistId")
				return plsRepo.Tracks(plsId, true)
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

func createPlaylistFromM3U(playlists core.Playlists) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		pls, err := playlists.ImportM3U(ctx, r.Body)
		if err != nil {
			log.Error(r.Context(), "Error parsing playlist", err)
			// TODO: consider returning StatusBadRequest for playlists that are malformed
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write([]byte(pls.ToM3U8()))
		if err != nil {
			log.Error(ctx, "Error sending m3u contents", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func handleExportPlaylist(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		plsRepo := ds.Playlist(ctx)
		plsId := chi.URLParam(r, "playlistId")
		pls, err := plsRepo.GetWithTracks(plsId, true)
		if errors.Is(err, model.ErrNotFound) {
			log.Warn(r.Context(), "Playlist not found", "playlistId", plsId)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Error(r.Context(), "Error retrieving the playlist", "playlistId", plsId, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Debug(ctx, "Exporting playlist as M3U", "playlistId", plsId, "name", pls.Name)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		disposition := fmt.Sprintf("attachment; filename=\"%s.m3u\"", pls.Name)
		w.Header().Set("Content-Disposition", disposition)

		_, err = w.Write([]byte(pls.ToM3U8()))
		if err != nil {
			log.Error(ctx, "Error sending playlist", "name", pls.Name)
			return
		}
	}
}

func deleteFromPlaylist(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := req.Params(r)
		playlistId, _ := p.String(":playlistId")
		ids, _ := p.Strings("id")
		err := ds.WithTx(func(tx model.DataStore) error {
			tracksRepo := tx.Playlist(r.Context()).Tracks(playlistId, true)
			return tracksRepo.Delete(ids...)
		})
		if len(ids) == 1 && errors.Is(err, model.ErrNotFound) {
			log.Warn(r.Context(), "Track not found in playlist", "playlistId", playlistId, "id", ids[0])
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Error(r.Context(), "Error deleting tracks from playlist", "playlistId", playlistId, "ids", ids, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var resp []byte
		if len(ids) == 1 {
			resp = []byte(`{"id":"` + ids[0] + `"}`)
		} else {
			resp, err = json.Marshal(&struct {
				Ids []string `json:"ids"`
			}{Ids: ids})
			if err != nil {
				log.Error(r.Context(), "Error marshaling delete response", "playlistId", playlistId, "ids", ids, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
		_, err = w.Write(resp)
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
		p := req.Params(r)
		playlistId, _ := p.String(":playlistId")
		var payload addTracksPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		tracksRepo := ds.Playlist(r.Context()).Tracks(playlistId, true)
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
		p := req.Params(r)
		playlistId, _ := p.String(":playlistId")
		id := p.IntOr(":id", 0)
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
		tracksRepo := ds.Playlist(r.Context()).Tracks(playlistId, true)
		err = tracksRepo.Reorder(id, newPos)
		if errors.Is(err, rest.ErrPermissionDenied) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
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
