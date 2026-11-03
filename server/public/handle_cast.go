package public

import (
	"errors"
	"net/http"
	"slices"
	"strconv"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/auth"
	streampkg "github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/req"
)

const maxCastStreamBitRate = 320

func (pub *Router) handleCastStream(w http.ResponseWriter, r *http.Request) {
	if !conf.Server.EnableExternalServices {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	ctx := r.Context()
	p := req.Params(r)
	tokenId, _ := p.String(":id")
	claims, err := decodeCastStreamClaims(tokenId)
	if err != nil {
		log.Warn(ctx, "Invalid cast stream token", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	owner, err := pub.ds.User().Get(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			log.Error(ctx, "Error retrieving user for cast stream", "userID", claims.UserID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}
	if err := auth.CheckClaims(claims, *owner, auth.AudienceCast); err != nil {
		log.Warn(ctx, "Rejected cast stream token", "userID", claims.UserID, err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx = request.WithUser(ctx, *owner)
	mf, err := pub.ds.MediaFile().Get(ctx, claims.ID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			log.Error(ctx, "Error retrieving media file for cast stream", "id", claims.ID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}
	if !owner.HasLibraryAccess(mf.LibraryID) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	streamReq := streampkg.Request{Format: claims.Format, BitRate: claims.BitRate}
	if pub.decider != nil {
		streamReq = pub.decider.ResolveRequest(ctx, mf, claims.Format, claims.BitRate, 0)
	}
	stream, err := pub.streamer.NewStream(ctx, mf, streamReq)
	if err != nil {
		if errors.Is(err, streampkg.ErrTooManyTranscodes) {
			w.Header().Set("Retry-After", strconv.Itoa(streampkg.RetryAfterSeconds))
			http.Error(w, "too many concurrent transcodes, please retry shortly", http.StatusTooManyRequests)
			return
		}
		log.Error(ctx, "Error starting cast stream", "id", claims.ID, err)
		http.Error(w, "invalid request", http.StatusInternalServerError)
		return
	}

	defer func() {
		if err := stream.Close(); err != nil && log.IsGreaterOrEqualTo(log.LevelDebug) {
			log.Error("Error closing cast stream", "id", claims.ID, "file", stream.Name(), err)
		}
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Content-Duration", strconv.FormatFloat(float64(stream.Duration()), 'G', -1, 32))

	n, err := stream.Serve(ctx, w, r)
	if err != nil || n == 0 {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func decodeCastStreamClaims(tokenString string) (auth.Claims, error) {
	c, err := auth.ValidatePublic(tokenString)
	if err != nil {
		return auth.Claims{}, err
	}
	if c.ExpiresAt.IsZero() {
		return auth.Claims{}, errors.New("cast stream token missing expiration")
	}
	if !slices.Contains(c.Audience, auth.AudienceCast) {
		return auth.Claims{}, auth.ErrWrongAudience
	}
	if c.ID == "" {
		return auth.Claims{}, errors.New("required claim \"id\" not found")
	}
	if c.UserID == "" {
		return auth.Claims{}, errors.New("required claim \"uid\" not found")
	}
	if c.Format != "" && c.Format != "raw" && c.Format != "mp3" {
		return auth.Claims{}, errors.New("unsupported cast stream format")
	}
	if c.BitRate < 0 || c.BitRate > maxCastStreamBitRate {
		return auth.Claims{}, errors.New("invalid cast stream bitrate")
	}
	return c, nil
}
