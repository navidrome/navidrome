package stream

import (
	"context"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// buildLegacyClientInfo translates legacy Subsonic stream/download parameters
// into a ClientInfo for use with MakeDecision.
func buildLegacyClientInfo(mf *model.MediaFile, reqFormat string, reqBitRate int, playerMaxBitRate int) *ClientInfo {
	ci := &ClientInfo{Name: "legacy"}

	// Determine target format for transcoding
	var targetFormat string
	switch {
	case reqFormat != "":
		targetFormat = reqFormat
	case reqBitRate > 0 && reqBitRate < mf.BitRate && conf.Server.DefaultDownsamplingFormat != "":
		targetFormat = conf.Server.DefaultDownsamplingFormat
	case playerMaxBitRate > 0 && playerMaxBitRate < mf.BitRate && conf.Server.DefaultDownsamplingFormat != "":
		// Server-side player MaxBitRate alone forces downsampling, even when the
		// client sent no format/bitrate params (issue #5583, legacy /stream path).
		targetFormat = conf.Server.DefaultDownsamplingFormat
	}

	if targetFormat != "" {
		// Add a direct play profile for the source format when no explicit
		// format was requested (bitrate-only downsampling) or when the
		// requested format matches the source. When the client explicitly
		// requests a different format, direct play must not match the
		// source — otherwise the source is returned untranscoded.
		if reqFormat == "" || strings.EqualFold(reqFormat, mf.Suffix) {
			ci.DirectPlayProfiles = []DirectPlayProfile{
				{Containers: []string{mf.Suffix}, AudioCodecs: []string{mf.AudioCodec()}, Protocols: []string{ProtocolHTTP}},
			}
		}
		ci.TranscodingProfiles = []Profile{
			{Container: targetFormat, AudioCodec: targetFormat, Protocol: ProtocolHTTP},
		}
		if reqBitRate > 0 {
			ci.MaxAudioBitrate = reqBitRate
			ci.MaxTranscodingAudioBitrate = reqBitRate
		}
	} else {
		// No transcoding requested — direct play everything
		ci.DirectPlayProfiles = []DirectPlayProfile{
			{Protocols: []string{ProtocolHTTP}},
		}
	}

	return ci
}

// ResolveRequest uses MakeDecision to resolve legacy Subsonic stream parameters
// into a fully specified Request.
func (s *deciderService) ResolveRequest(ctx context.Context, mf *model.MediaFile, reqFormat string, reqBitRate int, offset int) Request {
	var req Request
	req.Offset = offset

	if reqFormat == "raw" {
		req.Format = "raw"
		return req
	}

	playerMaxBitRate := 0
	if player, ok := request.PlayerFrom(ctx); ok {
		playerMaxBitRate = player.MaxBitRate
	}

	clientInfo := buildLegacyClientInfo(mf, reqFormat, reqBitRate, playerMaxBitRate)
	if resolved, ok := s.resolve(ctx, mf, clientInfo, offset); ok {
		return resolved
	}

	// No compatible profile for the requested format — retry with DefaultDownsamplingFormat
	// TODO: validate DefaultDownsamplingFormat at startup to warn about unsupported values
	fallbackFormat := conf.Server.DefaultDownsamplingFormat
	if reqFormat != "" && fallbackFormat != "" && !strings.EqualFold(reqFormat, fallbackFormat) {
		log.Warn(ctx, "Requested format not available, falling back to default downsampling format",
			"requestedFormat", reqFormat, "fallbackFormat", fallbackFormat, "id", mf.ID)
		return s.ResolveRequest(ctx, mf, fallbackFormat, reqBitRate, offset)
	}

	// Ultimate fallback — raw
	req.Format = "raw"
	return req
}

// ResolveClientRequest resolves a stream request for a client that declared its own direct play
// and transcoding profiles, falling back to raw when none fits.
func (s *deciderService) ResolveClientRequest(ctx context.Context, mf *model.MediaFile, clientInfo *ClientInfo, offset int) Request {
	if req, ok := s.resolve(ctx, mf, clientInfo, offset); ok {
		return req
	}
	return Request{Format: "raw", Offset: offset}
}

// resolve applies the server-side player overrides to clientInfo and maps the decision to a
// Request. ok is false when no profile fits.
func (s *deciderService) resolve(ctx context.Context, mf *model.MediaFile, clientInfo *ClientInfo, offset int) (Request, bool) {
	req := Request{Offset: offset}
	if trc, ok := request.TranscodingFrom(ctx); ok && trc.TargetFormat != "" {
		clientInfo = applyServerOverride(ctx, clientInfo, &trc)
	} else if player, ok := request.PlayerFrom(ctx); ok {
		modified := *clientInfo
		if modified.CapBitrate(player.MaxBitRate) {
			clientInfo = &modified
			log.Debug(ctx, "Applied player MaxBitRate cap", "playerMaxBitRate", player.MaxBitRate, "client", clientInfo.Name)
		}
	}

	decision, err := s.MakeDecision(ctx, mf, clientInfo, TranscodeOptions{SkipProbe: true})
	if err != nil {
		log.Error(ctx, "Error making transcode decision, falling back to raw", "id", mf.ID, err)
		req.Format = "raw"
		return req, true
	}
	switch {
	case decision.CanDirectPlay:
		req.Format = "raw"
	case decision.CanTranscode:
		req.Format = decision.TargetFormat
		req.BitRate = decision.TargetBitrate
		req.SampleRate = decision.TargetSampleRate
		req.BitDepth = decision.TargetBitDepth
		req.Channels = decision.TargetChannels
	default:
		return req, false
	}
	return req, true
}
