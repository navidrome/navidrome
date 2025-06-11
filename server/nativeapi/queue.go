package nativeapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/slice"
)

type updateQueuePayload struct {
	Ids      *[]string `json:"ids,omitempty"`
	Current  *int      `json:"current,omitempty"`
	Position *int64    `json:"position,omitempty"`
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
		var payload updateQueuePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		user, _ := request.UserFrom(ctx)
		client, _ := request.ClientFrom(ctx)
		ids := gg.V(payload.Ids)
		items := slice.Map(ids, func(id string) model.MediaFile {
			return model.MediaFile{ID: id}
		})
		current := gg.V(payload.Current)
		if len(ids) > 0 && (current < 0 || current >= len(ids)) {
			http.Error(w, "current index out of bounds", http.StatusBadRequest)
			return
		}
		pq := &model.PlayQueue{
			UserID:    user.ID,
			Current:   current,
			Position:  max(gg.V(payload.Position), 0),
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

func updateQueue(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var payload updateQueuePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		user, _ := request.UserFrom(ctx)
		client, _ := request.ClientFrom(ctx)
		pq := &model.PlayQueue{UserID: user.ID, ChangedBy: client}
		var cols []string

		if payload.Ids != nil {
			items := slice.Map(*payload.Ids, func(id string) model.MediaFile {
				return model.MediaFile{ID: id}
			})
			pq.Items = items
			cols = append(cols, "items")
		}

		if payload.Current != nil {
			pq.Current = *payload.Current
			cols = append(cols, "current")
			if payload.Ids != nil && len(*payload.Ids) > 0 && (*payload.Current < 0 || *payload.Current >= len(*payload.Ids)) {
				http.Error(w, "current index out of bounds", http.StatusBadRequest)
				return
			}
		}

		if payload.Position != nil {
			pq.Position = max(*payload.Position, 0)
			cols = append(cols, "position")
		}

		if len(cols) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if err := ds.PlayQueue(ctx).Store(pq, cols...); err != nil {
			log.Error(ctx, "Error updating queue", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func clearQueue(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)
		if err := ds.PlayQueue(ctx).Clear(user.ID); err != nil {
			log.Error(ctx, "Error clearing queue", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
