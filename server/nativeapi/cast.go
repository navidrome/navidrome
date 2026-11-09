package nativeapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

const (
	castTokenTTL        = 4 * time.Hour
	maxCastTokenBatch   = 50
	maxCastTokenBodyLen = 64 << 10
	maxCastBitRate      = 320
)

type castTokenRequestItem struct {
	Key        string `json:"key"`
	ID         string `json:"id"`
	IsRadio    bool   `json:"isRadio,omitempty"`
	Format     string `json:"format,omitempty"`
	MaxBitRate int    `json:"maxBitRate,omitempty"`
}

type castTokensRequest struct {
	Items []castTokenRequestItem `json:"items"`
}

type castTokenEntry struct {
	StreamToken string `json:"streamToken,omitempty"`
	CoverToken  string `json:"coverToken,omitempty"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type castTokensResponse struct {
	Tokens map[string]castTokenEntry `json:"tokens"`
}

func (api *Router) addCastRoute(r chi.Router) {
	r.Post("/cast/tokens", createCastTokens(api.ds))
}

func createCastTokens(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !conf.Server.EnableExternalServices {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		ctx := r.Context()
		user, ok := request.UserFrom(ctx)
		if !ok || user.ID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxCastTokenBodyLen)
		var payload castTokensRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(payload.Items) > maxCastTokenBatch {
			http.Error(w, "too many items", http.StatusBadRequest)
			return
		}

		now := time.Now().UTC()
		exp := now.Add(castTokenTTL)
		expUnix := exp.Unix()

		mfCache := make(map[string]*model.MediaFile, len(payload.Items))
		radioCache := make(map[string]*model.Radio)
		tokens := make(map[string]castTokenEntry, len(payload.Items))

		for _, item := range payload.Items {
			if item.ID == "" {
				continue
			}
			key := item.Key
			if key == "" {
				key = item.ID
			}

			if item.IsRadio {
				radio, cached := radioCache[item.ID]
				if !cached {
					fetched, err := ds.Radio().Get(ctx, item.ID)
					if err != nil {
						if !errors.Is(err, model.ErrNotFound) {
							log.Error(ctx, "Error loading radio for cast token", "id", item.ID, err)
						}
						radioCache[item.ID] = nil
						continue
					}
					radio = fetched
					radioCache[item.ID] = radio
				}
				if radio == nil || radio.UploadedImage == "" {
					continue
				}
				coverToken, err := auth.CreateExpiringPublicToken(exp, auth.Claims{
					IssuedAt: now,
					ID:       radio.CoverArtID().String(),
				})
				if err != nil {
					log.Error(ctx, "Error creating radio cast cover token", "id", item.ID, err)
					continue
				}
				tokens[key] = castTokenEntry{
					CoverToken: coverToken,
					ExpiresAt:  expUnix,
				}
				continue
			}

			format := item.Format
			if format == "" {
				format = "raw"
			}
			if format != "raw" && format != "mp3" {
				http.Error(w, "unsupported format", http.StatusBadRequest)
				return
			}
			if item.MaxBitRate < 0 || item.MaxBitRate > maxCastBitRate {
				http.Error(w, "invalid maxBitRate", http.StatusBadRequest)
				return
			}

			mf, cached := mfCache[item.ID]
			if !cached {
				fetched, err := ds.MediaFile().Get(ctx, item.ID)
				if err != nil {
					if !errors.Is(err, model.ErrNotFound) {
						log.Error(ctx, "Error loading media file for cast token", "id", item.ID, err)
					}
					mfCache[item.ID] = nil
					continue
				}
				mf = fetched
				mfCache[item.ID] = mf
			}
			if mf == nil || !user.HasLibraryAccess(mf.LibraryID) {
				continue
			}

			streamToken, err := auth.CreateExpiringPublicToken(exp, auth.Claims{
				Audience: []string{auth.AudienceCast},
				IssuedAt: now,
				UserID:   user.ID,
				Epoch:    user.TokenEpoch,
				ID:       mf.ID,
				Format:   format,
				BitRate:  item.MaxBitRate,
			})
			if err != nil {
				log.Error(ctx, "Error creating cast stream token", "id", mf.ID, err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			coverID := mf.CoverArtID().String()
			if coverID == "" {
				coverID = "mf-" + mf.ID
			}
			coverToken, err := auth.CreateExpiringPublicToken(exp, auth.Claims{
				IssuedAt: now,
				ID:       coverID,
			})
			if err != nil {
				log.Error(ctx, "Error creating cast cover token", "id", mf.ID, err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			tokens[key] = castTokenEntry{
				StreamToken: streamToken,
				CoverToken:  coverToken,
				ExpiresAt:   expUnix,
			}
		}

		resp, err := json.Marshal(castTokensResponse{Tokens: tokens})
		if err != nil {
			log.Error(ctx, "Error marshalling cast tokens", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(resp) //nolint:gosec
	}
}
