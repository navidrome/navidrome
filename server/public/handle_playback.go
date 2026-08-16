package public

import (
	"errors"
	"net/http"

	"github.com/navidrome/navidrome/core/auth"
	streampkg "github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/req"
)

func (pub *Router) handlePlayback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token, _ := req.Params(r).String(":id")
	claims, err := auth.ValidatePlaybackToken(token)
	if err != nil {
		log.Warn(ctx, "Error validating playback capability", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	mf, err := pub.ds.MediaFile(ctx).Get(claims.ID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			log.Error(ctx, "Error retrieving media file for playback capability", "id", claims.ID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	stream, err := pub.streamer.NewStream(ctx, mf, streampkg.Request{Format: "raw"})
	if err != nil {
		log.Error(ctx, "Error starting playback capability stream", "id", claims.ID, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := stream.Close(); err != nil && log.IsGreaterOrEqualTo(log.LevelDebug) {
			log.Error("Error closing playback capability stream", "id", claims.ID, "file", stream.Name(), err)
		}
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = stream.Serve(ctx, w, r)
}
