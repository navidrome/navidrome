package transcode

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// buildLegacyClientInfo translates legacy Subsonic stream/download parameters
// into a ClientInfo for use with MakeDecision.
// It does NOT read request.TranscodingFrom(ctx) — that is handled by
// MakeDecision's applyServerOverride.
func buildLegacyClientInfo(mf *model.MediaFile, reqFormat string, reqBitRate int) *ClientInfo {
	ci := &ClientInfo{Name: "legacy"}

	// Determine target format for transcoding
	var targetFormat string
	switch {
	case reqFormat != "":
		targetFormat = reqFormat
	case reqBitRate > 0 && reqBitRate < mf.BitRate && conf.Server.DefaultDownsamplingFormat != "":
		targetFormat = conf.Server.DefaultDownsamplingFormat
	}

	if targetFormat != "" {
		ci.DirectPlayProfiles = []DirectPlayProfile{
			{Containers: []string{mf.Suffix}, AudioCodecs: []string{mf.AudioCodec()}, Protocols: []string{ProtocolHTTP}},
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
// into a fully specified StreamRequest.
func (s *deciderService) ResolveRequest(ctx context.Context, mf *model.MediaFile, reqFormat string, reqBitRate int, offset int) StreamRequest {
	var req StreamRequest
	req.ID = mf.ID
	req.Offset = offset

	if reqFormat == "raw" {
		req.Format = "raw"
		return req
	}

	clientInfo := buildLegacyClientInfo(mf, reqFormat, reqBitRate)
	decision, err := s.MakeDecision(ctx, mf, clientInfo, DecisionOptions{SkipProbe: true})
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
