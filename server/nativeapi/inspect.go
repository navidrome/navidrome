package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/scanner/metadata_old"
	"github.com/navidrome/navidrome/utils/req"
)

type inspectOutput struct {
	File       string                  `json:"file"`
	RawTags    metadata_old.ParsedTags `json:"rawTags"`
	MappedTags model.MediaFile         `json:"mappedTags"`
}

func doInspect(ctx context.Context, ds model.DataStore, id string) (*inspectOutput, error) {
	file, err := ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	path := file.Path

	s, err := storage.For(file.LibraryPath)
	if err != nil {
		return nil, err
	}

	fs, err := s.FS()
	if err != nil {
		return nil, err
	}

	tags, err := fs.ReadTags(path)
	if err != nil {
		return nil, err
	}

	md := metadata.New(path, tags[path])
	result := &inspectOutput{
		File:       file.AbsolutePath(),
		RawTags:    tags[path].Tags,
		MappedTags: md.ToMediaFile(file.LibraryID, file.FolderID),
	}

	return result, nil
}

func inspect(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		p := req.Params(r)
		id, err := p.String("id")

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		output, err := doInspect(ctx, ds, id)
		if errors.Is(err, model.ErrNotFound) {
			log.Warn(ctx, "could not find file", "id", id)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if err != nil {
			log.Error(ctx, "Error reading tags", "id", id, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response, err := json.Marshal(output)
		if err != nil {
			log.Error(ctx, "Error marshalling json", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if _, err := w.Write(response); err != nil {
			log.Error(ctx, "Error sending response to client", err)
		}
	}
}
