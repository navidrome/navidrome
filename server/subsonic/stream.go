package subsonic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/transcode"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) serveStream(ctx context.Context, w http.ResponseWriter, r *http.Request, stream *core.Stream, id string) {
	if stream.Seekable() {
		http.ServeContent(w, r, stream.Name(), stream.ModTime(), stream)
	} else {
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
		} else {
			c, err := io.Copy(w, stream)
			if log.IsGreaterOrEqualTo(log.LevelDebug) {
				if err != nil {
					log.Error(ctx, "Error sending transcoded file", "id", id, err)
				} else {
					log.Trace(ctx, "Success sending transcode file", "id", id, "size", c)
				}
			}
		}
	}
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

	streamReq := api.resolveStreamRequest(ctx, mf, format, maxBitRate, timeOffset)
	stream, err := api.streamer.DoStream(ctx, mf, streamReq)
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

	api.serveStream(ctx, w, r, stream, id)

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
		streamReq := api.resolveStreamRequest(ctx, v, format, maxBitRate, 0)
		stream, err := api.streamer.DoStream(ctx, v, streamReq)
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

		api.serveStream(ctx, w, r, stream, id)
		return nil, nil
	case *model.Album:
		setHeaders(v.Name)
		err = api.archiver.ZipAlbum(ctx, id, format, maxBitRate, w)
	case *model.Artist:
		setHeaders(v.Name)
		err = api.archiver.ZipArtist(ctx, id, format, maxBitRate, w)
	case *model.Playlist:
		setHeaders(v.Name)
		err = api.archiver.ZipPlaylist(ctx, id, format, maxBitRate, w)
	default:
		err = model.ErrNotFound
	}

	return nil, err
}

// buildLegacyClientInfo translates legacy Subsonic stream/download parameters
// into a transcode.ClientInfo for use with MakeDecision.
// It does NOT read request.TranscodingFrom(ctx) — that is handled by
// MakeDecision's applyServerOverride.
func buildLegacyClientInfo(mf *model.MediaFile, reqFormat string, reqBitRate int) *transcode.ClientInfo {
	ci := &transcode.ClientInfo{Name: "legacy"}

	// Determine target format for transcoding
	var targetFormat string
	switch {
	case reqFormat != "":
		targetFormat = reqFormat
	case reqBitRate > 0 && reqBitRate < mf.BitRate && conf.Server.DefaultDownsamplingFormat != "":
		targetFormat = conf.Server.DefaultDownsamplingFormat
	}

	if targetFormat != "" {
		ci.DirectPlayProfiles = []transcode.DirectPlayProfile{
			{Containers: []string{mf.Suffix}, AudioCodecs: []string{mf.AudioCodec()}, Protocols: []string{transcode.ProtocolHTTP}},
		}
		ci.TranscodingProfiles = []transcode.Profile{
			{Container: targetFormat, AudioCodec: targetFormat, Protocol: transcode.ProtocolHTTP},
		}
		if reqBitRate > 0 {
			ci.MaxAudioBitrate = reqBitRate
			ci.MaxTranscodingAudioBitrate = reqBitRate
		}
	} else {
		// No transcoding requested — direct play everything
		ci.DirectPlayProfiles = []transcode.DirectPlayProfile{
			{Protocols: []string{transcode.ProtocolHTTP}},
		}
	}

	return ci
}

// resolveStreamRequest uses MakeDecision to resolve legacy stream parameters
// into a fully specified StreamRequest.
func (api *Router) resolveStreamRequest(ctx context.Context, mf *model.MediaFile, reqFormat string, reqBitRate int, offset int) core.StreamRequest {
	req := core.StreamRequest{ID: mf.ID, Offset: offset}

	if reqFormat == "raw" {
		req.Format = "raw"
		return req
	}

	clientInfo := buildLegacyClientInfo(mf, reqFormat, reqBitRate)
	decision, err := api.transcodeDecision.MakeDecision(ctx, mf, clientInfo, transcode.DecisionOptions{SkipProbe: true})
	if err != nil {
		log.Error(ctx, "Error making transcode decision, falling back to raw", "id", mf.ID, err)
		req.Format = "raw"
		return req
	}

	if decision.CanDirectPlay {
		req.Format = "raw"
		return req
	}

	if decision.CanTranscode {
		req.Format = decision.TargetFormat
		req.BitRate = decision.TargetBitrate
		req.SampleRate = decision.TargetSampleRate
		req.BitDepth = decision.TargetBitDepth
		req.Channels = decision.TargetChannels
		return req
	}

	// No compatible profile — fallback to raw
	req.Format = "raw"
	return req
}
