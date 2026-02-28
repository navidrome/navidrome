package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/req"
	_ "golang.org/x/image/webp"
)

type restHandler = func(rest.RepositoryConstructor, ...rest.Logger) http.HandlerFunc

func playlistTracksHandler(pls playlists.Playlists, handler restHandler, refreshSmartPlaylist func(*http.Request) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plsId := chi.URLParam(r, "playlistId")
		tracks := pls.TracksRepository(r.Context(), plsId, refreshSmartPlaylist(r))
		if tracks == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		handler(func(ctx context.Context) rest.Repository { return tracks }).ServeHTTP(w, r)
	}
}

func getPlaylist(pls playlists.Playlists) http.HandlerFunc {
	handler := playlistTracksHandler(pls, rest.GetAll, func(r *http.Request) bool {
		return req.Params(r).Int64Or("_start", 0) == 0
	})
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Header.Get("accept")) == "audio/x-mpegurl" {
			handleExportPlaylist(pls)(w, r)
			return
		}
		handler(w, r)
	}
}

func getPlaylistTrack(pls playlists.Playlists) http.HandlerFunc {
	return playlistTracksHandler(pls, rest.Get, func(*http.Request) bool { return true })
}

func createPlaylistFromM3U(pls playlists.Playlists) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		pl, err := pls.ImportM3U(ctx, r.Body)
		if err != nil {
			log.Error(r.Context(), "Error parsing playlist", err)
			// TODO: consider returning StatusBadRequest for playlists that are malformed
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write([]byte(pl.ToM3U8())) //nolint:gosec
		if err != nil {
			log.Error(ctx, "Error sending m3u contents", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func handleExportPlaylist(pls playlists.Playlists) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		plsId := chi.URLParam(r, "playlistId")
		playlist, err := pls.GetWithTracks(ctx, plsId)
		if errors.Is(err, model.ErrNotFound) {
			log.Warn(ctx, "Playlist not found", "playlistId", plsId)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Error(ctx, "Error retrieving the playlist", "playlistId", plsId, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Debug(ctx, "Exporting playlist as M3U", "playlistId", plsId, "name", playlist.Name)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		disposition := fmt.Sprintf("attachment; filename=\"%s.m3u\"", playlist.Name)
		w.Header().Set("Content-Disposition", disposition)

		_, err = w.Write([]byte(playlist.ToM3U8())) //nolint:gosec
		if err != nil {
			log.Error(ctx, "Error sending playlist", "name", playlist.Name)
			return
		}
	}
}

func deleteFromPlaylist(pls playlists.Playlists) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := req.Params(r)
		playlistId, _ := p.String(":playlistId")
		ids, _ := p.Strings("id")
		err := pls.RemoveTracks(r.Context(), playlistId, ids)
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
		writeDeleteManyResponse(w, r, ids)
	}
}

func addToPlaylist(pls playlists.Playlists) http.HandlerFunc {
	type addTracksPayload struct {
		Ids       []string       `json:"ids"`
		AlbumIds  []string       `json:"albumIds"`
		ArtistIds []string       `json:"artistIds"`
		Discs     []model.DiscID `json:"discs"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		p := req.Params(r)
		playlistId, _ := p.String(":playlistId")
		var payload addTracksPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		count, c := 0, 0
		if c, err = pls.AddTracks(ctx, playlistId, payload.Ids); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		count += c
		if c, err = pls.AddAlbums(ctx, playlistId, payload.AlbumIds); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		count += c
		if c, err = pls.AddArtists(ctx, playlistId, payload.ArtistIds); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		count += c
		if c, err = pls.AddDiscs(ctx, playlistId, payload.Discs); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		count += c

		// Must return an object with an ID, to satisfy ReactAdmin `create` call
		_, err = fmt.Fprintf(w, `{"added":%d}`, count) //nolint:gosec
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func reorderItem(pls playlists.Playlists) http.HandlerFunc {
	type reorderPayload struct {
		InsertBefore string `json:"insert_before"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
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
		err = pls.ReorderTrack(ctx, playlistId, id, newPos)
		if errors.Is(err, model.ErrNotAuthorized) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err = w.Write(fmt.Appendf(nil, `{"id":"%d"}`, id)) //nolint:gosec
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func getSongPlaylists(svc playlists.Playlists) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := req.Params(r)
		trackId, _ := p.String(":id")
		playlists, err := svc.GetPlaylists(r.Context(), trackId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := json.Marshal(playlists)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(data) //nolint:gosec
	}
}

const maxImageSize = 10 << 20 // 10MB

func uploadPlaylistImage(pls playlists.Playlists) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		p := req.Params(r)
		playlistId, _ := p.String(":id")

		if err := r.ParseMultipartForm(maxImageSize); err != nil {
			log.Error(ctx, "Error parsing multipart form", err)
			http.Error(w, "file too large or invalid form", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("image")
		if err != nil {
			log.Error(ctx, "Error reading uploaded file", err)
			http.Error(w, "missing image file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Validate the uploaded file is a valid image
		_, format, err := image.DecodeConfig(file)
		if err != nil {
			log.Error(ctx, "Uploaded file is not a valid image", err)
			http.Error(w, "invalid image file", http.StatusBadRequest)
			return
		}

		// Reset reader after DecodeConfig consumed some bytes
		if seeker, ok := file.(io.Seeker); ok {
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				log.Error(ctx, "Error seeking file", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Determine file extension from decoded format or original filename
		ext := "." + format
		if ext == "." {
			ext = strings.ToLower(filepath.Ext(header.Filename))
		}
		if ext == "" || ext == "." {
			log.Error(ctx, "Could not determine image type", "playlistId", playlistId, "filename", header.Filename)
			http.Error(w, "could not determine image type", http.StatusBadRequest)
			return
		}

		err = pls.SetImage(ctx, playlistId, file, ext)
		if errors.Is(err, model.ErrNotAuthorized) {
			log.Error(ctx, "Not authorized to upload playlist image", "playlistId", playlistId, err)
			http.Error(w, "not authorized", http.StatusForbidden)
			return
		}
		if errors.Is(err, model.ErrNotFound) {
			log.Error(ctx, "Playlist not found for image upload", "playlistId", playlistId, err)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Error(ctx, "Error saving playlist image", "playlistId", playlistId, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, _ = fmt.Fprintf(w, `{"status":"ok"}`) //nolint:gosec
	}
}

func deletePlaylistImage(pls playlists.Playlists) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		p := req.Params(r)
		playlistId, _ := p.String(":id")

		err := pls.RemoveImage(ctx, playlistId)
		if errors.Is(err, model.ErrNotAuthorized) {
			log.Error(ctx, "Not authorized to remove playlist image", "playlistId", playlistId, err)
			http.Error(w, "not authorized", http.StatusForbidden)
			return
		}
		if errors.Is(err, model.ErrNotFound) {
			log.Error(ctx, "Playlist not found for image removal", "playlistId", playlistId, err)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Error(ctx, "Error removing playlist image", "playlistId", playlistId, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, _ = fmt.Fprintf(w, `{"status":"ok"}`) //nolint:gosec
	}
}
