package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tagwriter"
	"github.com/Masterminds/squirrel"
)

type AlbumUpdateRequest struct {
	Album        string `json:"album"`
	Name         string `json:"name"`
	AlbumArtist  string `json:"albumArtist"`
	Year         *int   `json:"year"`
	Genre        string `json:"genre"`
	Comment      string `json:"comment"`
}

func (api *Router) addAlbumRoute(r chi.Router) {
	albumConstructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model.Album{})
	}

	r.Route("/album", func(r chi.Router) {
		r.Get("/", rest.GetAll(albumConstructor))

		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(albumConstructor))
			r.Put("/", api.updateAlbum())
		})
	})
}

func (api *Router) updateAlbum() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		bodyBytesDebug, _ := io.ReadAll(r.Body)
		log.Info(r.Context(), "DEBUG: Raw JSON Received", "json", string(bodyBytesDebug))
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytesDebug))

		if !conf.Server.EnableTagEditing {
			log.Warn(r.Context(), "Tag editing attempt while disabled")
			http.Error(w, "Tag editing is disabled in configuration", http.StatusForbidden)
			return
		}

		albumID := chi.URLParamFromCtx(ctx, "id")
		if albumID == "" {
			log.Warn(r.Context(), "Album ID missing in update request")
			http.Error(w, "Album ID is required", http.StatusBadRequest)
			return
		}

		log.Debug(r.Context(), "Fetching Album", "id", albumID)
		album, err := api.ds.Album(ctx).Get(albumID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				log.Warn(r.Context(), "Album not found", "id", albumID)
				http.Error(w, "Album not found", http.StatusNotFound)
				return
			}
			log.Error(r.Context(), "Failed to retrieve album", "error", err, "id", albumID)
			http.Error(w, "Failed to retrieve album", http.StatusInternalServerError)
			return
		}

		log.Debug(r.Context(), "Album retrieved", "album_id", album.ID, "name", album.Name, "song_count", album.SongCount)

		log.Debug(r.Context(), "Fetching MediaFiles for album", "albumId", albumID)
		mediaFiles, err := api.ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_id": albumID}})
		if err != nil {
			log.Error(r.Context(), "Failed to retrieve media files for album", "error", err, "albumId", albumID)
			http.Error(w, "Failed to retrieve media files", http.StatusInternalServerError)
			return
		}

		log.Info(r.Context(), "Batch update starting", "album_id", albumID, "count", len(mediaFiles))

		if len(mediaFiles) == 0 {
			log.Warn(r.Context(), "No media files found for album", "albumId", albumID)
			http.Error(w, "No media files found for this album", http.StatusNotFound)
			return
		}

		log.Debug(r.Context(), "Parsing request body", "albumId", albumID)

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error(r.Context(), "Failed to read request body", "error", err)
			http.Error(w, "Failed to read request", http.StatusBadRequest)
			return
		}
		log.Debug(r.Context(), "Raw request body", "body", string(bodyBytes))

		var req AlbumUpdateRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			log.Error(r.Context(), "Failed to decode JSON payload", "error", err)
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		log.Debug(r.Context(), "Request body parsed", "album", req.Album, "albumArtist", req.AlbumArtist, "year", req.Year)

		newAlbumName := req.Album
		newArtist := req.AlbumArtist
		newYear := req.Year
		newGenre := req.Genre
		newComment := req.Comment

		log.Debug(r.Context(), "Local variables assigned", "newAlbumName", newAlbumName, "newArtist", newArtist)

		titleToUse := newAlbumName
		if titleToUse == "" {
			titleToUse = req.Name
		}
		log.Info(r.Context(), "DEBUG: Title to be used for tracks", "title", titleToUse)

		tw := tagwriter.New()
		updatedCount := 0
		failedCount := 0

		for _, mf := range mediaFiles {
			absPath := mf.AbsolutePath()

			log.Info(r.Context(), "Processing track", "mediaFileId", mf.ID, "path", absPath, "newAlbum", titleToUse)

			tags := make(tagwriter.Tags)
			tags[tagwriter.TagAlbum] = titleToUse
			tags[tagwriter.TagAlbumArtist] = newArtist
			if newYear != nil && *newYear > 0 {
				tags[tagwriter.TagYear] = strconv.Itoa(*newYear)
			}
			if newGenre != "" {
				tags[tagwriter.TagGenre] = newGenre
			}
			if newComment != "" {
				tags[tagwriter.TagComment] = newComment
			}

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
				log.Error(r.Context(), "Failed to write tags to file", "error", err, "path", absPath, "mediaFileId", mf.ID)
				failedCount++
				continue
			}

			if err := os.Chtimes(absPath, time.Now(), time.Now()); err != nil {
				log.Error(r.Context(), "Failed to update file modification time", "error", err, "path", absPath)
			}

			mf.Album = titleToUse
			mf.AlbumArtist = newArtist
			if newYear != nil && *newYear > 0 {
				mf.Year = *newYear
			}
			mf.Genre = newGenre
			mf.Comment = newComment

			log.Debug(r.Context(), "Updating MediaFile record", "mediaFileId", mf.ID, "album", mf.Album, "albumArtist", mf.AlbumArtist)
			if err := api.ds.MediaFile(ctx).Put(&mf); err != nil {
				log.Error(r.Context(), "Failed to update MediaFile in database", "error", err, "mediaFileId", mf.ID)
				failedCount++
				continue
			}

			updatedCount++
			log.Debug(r.Context(), "Successfully updated media file", "mediaFileId", mf.ID)
		}

		if req.Album != "" {
			album.Name = req.Album
		}
		if req.AlbumArtist != "" {
			album.AlbumArtist = req.AlbumArtist
		}
		if req.Year != nil && *req.Year > 0 {
			album.MaxYear = *req.Year
		}
		if req.Genre != "" {
			album.Genre = req.Genre
		}
		if req.Comment != "" {
			album.Comment = req.Comment
		}

		if err := api.ds.Album(ctx).Put(album); err != nil {
			log.Error(r.Context(), "Failed to update Album in database", "error", err, "albumId", albumID)
			http.Error(w, "Failed to update album", http.StatusInternalServerError)
			return
		}

		log.Info(r.Context(), "Album batch update completed", "albumId", albumID, "updated", updatedCount, "failed", failedCount)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"` + albumID + `", "name":"` + album.Name + `", "updated":` + strconv.Itoa(updatedCount) + `, "failed":` + strconv.Itoa(failedCount) + `}`))
	}
}