package public

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
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

	stream, err := pub.streamer.NewStream(ctx, info.id, info.format, info.bitrate, 0)
	if err != nil {
		log.Error(ctx, "Error starting shared stream", err)
		http.Error(w, "invalid request", http.StatusInternalServerError)
	}

	// Make sure the stream will be closed at the end, to avoid leakage
	defer func() {
		if err := stream.Close(); err != nil && log.IsGreaterOrEqualTo(log.LevelDebug) {
			log.Error("Error closing shared stream", "id", info.id, "file", stream.Name(), err)
		}
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Content-Duration", strconv.FormatFloat(float64(stream.Duration()), 'G', -1, 32))

	if stream.Seekable() {
		http.ServeContent(w, r, stream.Name(), stream.ModTime(), stream)
	} else {
		// If the stream doesn't provide a size (i.e. is not seekable), we can't support ranges/content-length
		w.Header().Set("Accept-Ranges", "none")
		w.Header().Set("Content-Type", stream.ContentType())

		estimateContentLength := p.BoolOr("estimateContentLength", false)

		// if Client requests the estimated content-length, send it
		if estimateContentLength {
			length := strconv.Itoa(stream.EstimatedContentLength())
			log.Trace(ctx, "Estimated content-length", "contentLength", length)
			w.Header().Set("Content-Length", length)
		}

		if r.Method == http.MethodHead {
			go func() { _, _ = io.Copy(io.Discard, stream) }()
		} else {
			c, err := io.Copy(w, stream)
			if log.IsGreaterOrEqualTo(log.LevelDebug) {
				if err != nil {
					log.Error(ctx, "Error sending shared transcoded file", "id", info.id, err)
				} else {
					log.Trace(ctx, "Success sending shared transcode file", "id", info.id, "size", c)
				}
			}
		}
	}
}

type shareTrackInfo struct {
	id      string
	format  string
	bitrate int
}

func decodeStreamInfo(tokenString string) (shareTrackInfo, error) {
	token, err := auth.TokenAuth.Decode(tokenString)
	if err != nil {
		return shareTrackInfo{}, err
	}
	if token == nil {
		return shareTrackInfo{}, errors.New("unauthorized")
	}
	err = jwt.Validate(token, jwt.WithRequiredClaim("id"))
	if err != nil {
		return shareTrackInfo{}, err
	}
	claims, err := token.AsMap(context.Background())
	if err != nil {
		return shareTrackInfo{}, err
	}
	id, ok := claims["id"].(string)
	if !ok {
		return shareTrackInfo{}, errors.New("invalid id type")
	}
	resp := shareTrackInfo{}
	resp.id = id
	resp.format, _ = claims["f"].(string)
	resp.bitrate, _ = claims["b"].(int)
	return resp, nil
}
