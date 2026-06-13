package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tagwriter"
)

type SongUpdateRequest struct {
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"albumArtist"`
	Year        *int   `json:"year"`
	Genre       string `json:"genre"`
	TrackNumber *int  `json:"trackNumber"`
}

func (api *Router) addSongRoute(r chi.Router) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model.MediaFile{})
	}

	r.Route("/song", func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Method not allowed"}`))
		})
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			r.Put("/", api.updateSong())
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				w.Write([]byte(`{"error": "Method not allowed"}`))
			})
		})
	})
}

func (api *Router) updateSong() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !conf.Server.EnableTagEditing {
			log.Warn(r.Context(), "Tag editing attempt while disabled")
			http.Error(w, "Tag editing is disabled in configuration", http.StatusForbidden)
			return
		}

		songID := chi.URLParamFromCtx(ctx, "id")
		if songID == "" {
			log.Warn(r.Context(), "Song ID missing in update request")
			http.Error(w, "Song ID is required", http.StatusBadRequest)
			return
		}

		log.Debug(r.Context(), "Fetching MediaFile", "id", songID)
		mf, err := api.ds.MediaFile(ctx).Get(songID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				log.Warn(r.Context(), "Song not found", "id", songID)
				http.Error(w, "Song not found", http.StatusNotFound)
				return
			}
			log.Error(r.Context(), "Failed to retrieve song", "error", err, "id", songID)
			http.Error(w, "Failed to retrieve song", http.StatusInternalServerError)
			return
		}

		log.Debug(r.Context(), "Parsing request body", "id", songID)
		var req SongUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error(r.Context(), "Failed to decode JSON payload", "error", err)
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		log.Debug(r.Context(), "Writing tags to file", "path", mf.AbsolutePath(), "id", songID)
		absPath := mf.AbsolutePath()

		tags := make(tagwriter.Tags)
		if req.Title != "" {
			tags[tagwriter.TagTitle] = req.Title
		}
		if req.Artist != "" {
			tags[tagwriter.TagArtist] = req.Artist
		}
		if req.Album != "" {
			tags[tagwriter.TagAlbum] = req.Album
		}
		if req.AlbumArtist != "" {
			tags[tagwriter.TagAlbumArtist] = req.AlbumArtist
		}
		if req.Year != nil && *req.Year > 0 {
			tags[tagwriter.TagYear] = strconv.Itoa(*req.Year)
		}
		if req.Genre != "" {
			tags[tagwriter.TagGenre] = req.Genre
		}
		if req.TrackNumber != nil && *req.TrackNumber > 0 {
			tags[tagwriter.TagTrackNumber] = strconv.Itoa(*req.TrackNumber)
		}

		tw := tagwriter.New()
		if err := tw.WriteTags(absPath, tags); err != nil {
			if errors.Is(err, tagwriter.ErrFeatureDisabled) {
				log.Warn(r.Context(), "Tag writing disabled in config", "error", err)
				http.Error(w, "Tag editing is disabled in configuration", http.StatusForbidden)
				return
			}
			if errors.Is(err, tagwriter.ErrUnsupportedFormat) {
				log.Warn(r.Context(), "Unsupported file format", "error", err, "path", absPath)
				http.Error(w, "Unsupported file format", http.StatusBadRequest)
				return
			}
			if errors.Is(err, tagwriter.ErrReadOnlyFile) {
				log.Warn(r.Context(), "File is read-only", "error", err, "path", absPath)
				http.Error(w, "File is read-only", http.StatusForbidden)
				return
			}
			log.Error(r.Context(), "Failed to write tags", "error", err, "path", absPath)
			http.Error(w, "Failed to write tags: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Debug(r.Context(), "Updating MediaFile in database", "id", songID)
		if req.Title != "" {
			mf.Title = req.Title
		}
		if req.Artist != "" {
			mf.Artist = req.Artist
		}
		if req.Album != "" {
			mf.Album = req.Album
		}
		if req.AlbumArtist != "" {
			mf.AlbumArtist = req.AlbumArtist
		}
		if req.Year != nil && *req.Year > 0 {
			mf.Year = *req.Year
		}
		if req.Genre != "" {
			mf.Genre = req.Genre
		}
		if req.TrackNumber != nil && *req.TrackNumber > 0 {
			mf.TrackNumber = *req.TrackNumber
		}

		if err := api.ds.MediaFile(ctx).Put(mf); err != nil {
			log.Error(r.Context(), "Failed to update database", "error", err, "id", songID)
			http.Error(w, "Failed to update database", http.StatusInternalServerError)
			return
		}

		log.Info(r.Context(), "Song updated successfully", "id", songID, "title", mf.Title)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"` + songID + `", "title":"` + mf.Title + `"}`))
	}
}