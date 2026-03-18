package subsonic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) serveStream(ctx context.Context, w http.ResponseWriter, r *http.Request, stream *stream.Stream, id string) error {
	if stream.Seekable() {
		http.ServeContent(w, r, stream.Name(), stream.ModTime(), stream)
		return nil
	}

	// If the stream doesn't provide a size (i.e. is not seekable), we can't support ranges/content-length
	w.Header().Set("Accept-Ranges", "none")
	w.Header().Set("Content-Type", stream.ContentType())

	estimateContentLength := req.Params(r).BoolOr("estimateContentLength", false)

	// if Client requests the estimated content-length, send it
	if estimateContentLength {
		length := strconv.Itoa(stream.EstimatedContentLength())
		log.Trace(ctx, "Estimated content-length", "contentLength", length)
		w.Header().Set("Content-Length", length)
	}

	if r.Method == http.MethodHead {
		go func() { _, _ = io.Copy(io.Discard, stream) }()
		return nil
	}

	c, err := io.Copy(w, stream)
	if err != nil {
		log.Error(ctx, "Error sending transcoded file", "id", id, err)
		if c == 0 {
			// No bytes written yet, safe to send an error response
			w.Header().Del("Content-Length")
			return fmt.Errorf("sending transcoded file: %w", err)
		}
		// Bytes already written (200 committed), can't change the response
		return nil
	}
	if c == 0 {
		log.Error(ctx, "Transcoding returned empty output, ffmpeg may have failed. "+
			"Check that ffmpeg supports the requested codec. Enable Trace logging for ffmpeg stderr details",
			"id", id, "format", stream.ContentType())
		w.Header().Del("Content-Length")
		return errors.New("transcoding failed: empty output")
	}
	log.Trace(ctx, "Success sending transcoded file", "id", id, "size", c)
	return nil
}

func (api *Router) Stream(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	maxBitRate := p.IntOr("maxBitRate", 0)
	format, _ := p.String("format")
	timeOffset := p.IntOr("timeOffset", 0)

	mf, err := api.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	streamReq := api.transcodeDecision.ResolveRequest(ctx, mf, format, maxBitRate, timeOffset)
	stream, err := api.streamer.NewStream(ctx, mf, streamReq)
	if err != nil {
		return nil, err
	}

	// Make sure the stream will be closed at the end, to avoid leakage
	defer func() {
		if err := stream.Close(); err != nil && log.IsGreaterOrEqualTo(log.LevelDebug) {
			log.Error("Error closing stream", "id", id, "file", stream.Name(), err)
		}
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Content-Duration", strconv.FormatFloat(float64(stream.Duration()), 'G', -1, 32))

	if err := api.serveStream(ctx, w, r, stream, id); err != nil {
		return nil, err
	}

	return nil, nil
}

func (api *Router) Download(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	username, _ := request.UsernameFrom(ctx)
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	if !conf.Server.EnableDownloads {
		log.Warn(ctx, "Downloads are disabled", "user", username, "id", id)
		return nil, newError(responses.ErrorAuthorizationFail, "downloads are disabled")
	}

	entity, err := model.GetEntityByID(ctx, api.ds, id)
	if err != nil {
		return nil, err
	}

	maxBitRate := p.IntOr("bitrate", 0)
	format, _ := p.String("format")

	if format == "" {
		if conf.Server.AutoTranscodeDownload {
			// if we are not provided a format, see if we have requested transcoding for this client
			// This must be enabled via a config option. For the UI, we are always given an option.
			// This will impact other clients which do not use the UI
			transcoding, ok := request.TranscodingFrom(ctx)

			if !ok {
				format = "raw"
			} else {
				format = transcoding.TargetFormat
				maxBitRate = transcoding.DefaultBitRate
			}
		} else {
			format = "raw"
		}
	}

	setHeaders := func(name string) {
		name = strings.ReplaceAll(name, ",", "_")
		disposition := fmt.Sprintf("attachment; filename=\"%s.zip\"", name)
		w.Header().Set("Content-Disposition", disposition)
		w.Header().Set("Content-Type", "application/zip")
	}

	switch v := entity.(type) {
	case *model.MediaFile:
		streamReq := api.transcodeDecision.ResolveRequest(ctx, v, format, maxBitRate, 0)
		stream, err := api.streamer.NewStream(ctx, v, streamReq)
		if err != nil {
			return nil, err
		}

		// Make sure the stream will be closed at the end, to avoid leakage
		defer func() {
			if err := stream.Close(); err != nil && log.IsGreaterOrEqualTo(log.LevelDebug) {
				log.Error("Error closing stream", "id", id, "file", stream.Name(), err)
			}
		}()

		disposition := fmt.Sprintf("attachment; filename=\"%s\"", stream.Name())
		w.Header().Set("Content-Disposition", disposition)

		return nil, api.serveStream(ctx, w, r, stream, id)
	case *model.Album:
		setHeaders(v.Name)
		return nil, api.archiver.ZipAlbum(ctx, id, format, maxBitRate, w)
	case *model.Artist:
		setHeaders(v.Name)
		return nil, api.archiver.ZipArtist(ctx, id, format, maxBitRate, w)
	case *model.Playlist:
		setHeaders(v.Name)
		return nil, api.archiver.ZipPlaylist(ctx, id, format, maxBitRate, w)
	default:
		return nil, model.ErrNotFound
	}
}
