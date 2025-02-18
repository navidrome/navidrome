package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/req"
)

func doInspect(ctx context.Context, ds model.DataStore, id string) (*core.InspectOutput, error) {
	file, err := ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	if file.Missing {
		return nil, model.ErrNotFound
	}

	return core.Inspect(file.AbsolutePath(), file.LibraryID, file.FolderID)
}

func inspect(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		user, _ := request.UserFrom(ctx)
		if !user.IsAdmin {
			http.Error(w, "Inspect is only available to admin users", http.StatusUnauthorized)
		}

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

		output.MappedTags = nil
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
