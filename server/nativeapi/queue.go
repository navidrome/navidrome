package nativeapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/slice"
)

type queuePayload struct {
	Ids      []string `json:"ids"`
	Current  int      `json:"current"`
	Position int64    `json:"position"`
}

func getQueue(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)
		repo := ds.PlayQueue(ctx)
		pq, err := repo.Retrieve(user.ID)
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			log.Error(ctx, "Error retrieving queue", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if pq == nil {
			pq = &model.PlayQueue{}
		}
		resp, err := json.Marshal(pq)
		if err != nil {
			log.Error(ctx, "Error marshalling queue", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(resp)
	}
}

func saveQueue(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var payload queuePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		user, _ := request.UserFrom(ctx)
		client, _ := request.ClientFrom(ctx)
		items := slice.Map(payload.Ids, func(id string) model.MediaFile {
			return model.MediaFile{ID: id}
		})
		if len(payload.Ids) > 0 && (payload.Current < 0 || payload.Current >= len(payload.Ids)) {
			http.Error(w, "current index out of bounds", http.StatusBadRequest)
			return
		}
		pq := &model.PlayQueue{
			UserID:    user.ID,
			Current:   payload.Current,
			Position:  payload.Position,
			ChangedBy: client,
			Items:     items,
		}
		if err := ds.PlayQueue(ctx).Store(pq); err != nil {
			log.Error(ctx, "Error saving queue", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
