package public

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	. "github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/req"
)

func (pub *Router) handleStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := req.Params(r)
	tokenId, _ := p.String(":id")
	info, err := decodeStreamInfo(tokenId)
	if err != nil {
		log.Error(ctx, "Error parsing shared stream info", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if info.shareID != "" {
		share, err := pub.ds.Share(ctx).Get(info.shareID)
		if err != nil {
			checkShareError(ctx, w, err, info.shareID)
			return
		}
		if expiresAt := V(share.ExpiresAt); !expiresAt.IsZero() && expiresAt.Before(time.Now()) {
			checkShareError(ctx, w, model.ErrExpired, info.shareID)
			return
		}
	}

	mf, err := pub.ds.MediaFile(ctx).Get(info.id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			log.Error(ctx, "Error retrieving media file for shared stream", "id", info.id, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	stream, err := pub.streamer.NewStream(ctx, mf, stream.Request{
		Format: info.format, BitRate: info.bitrate,
	})
	if err != nil {
		log.Error(ctx, "Error starting shared stream", err)
		http.Error(w, "invalid request", http.StatusInternalServerError)
		return
	}

	// Make sure the stream will be closed at the end, to avoid leakage
	defer func() {
		if err := stream.Close(); err != nil && log.IsGreaterOrEqualTo(log.LevelDebug) {
			log.Error("Error closing shared stream", "id", info.id, "file", stream.Name(), err)
		}
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Content-Duration", strconv.FormatFloat(float64(stream.Duration()), 'G', -1, 32))

	n, err := stream.Serve(ctx, w, r)
	if err != nil || n == 0 {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

type shareTrackInfo struct {
	id      string
	format  string
	bitrate int
	shareID string
}

func decodeStreamInfo(tokenString string) (shareTrackInfo, error) {
	c, err := auth.Validate(tokenString)
	if err != nil {
		return shareTrackInfo{}, err
	}
	if c.ID == "" {
		return shareTrackInfo{}, errors.New("required claim \"id\" not found")
	}
	return shareTrackInfo{
		id:      c.ID,
		format:  c.Format,
		bitrate: c.BitRate,
		shareID: c.ShareID,
	}, nil
}
