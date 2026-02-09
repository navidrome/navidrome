package transcode

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const (
	tokenTTL       = 12 * time.Hour
	defaultBitrate = 256 // kbps
)

func NewDecider(ds model.DataStore) Decider {
	return &deciderService{
		ds: ds,
	}
}

type deciderService struct {
	ds model.DataStore
}

func (s *deciderService) MakeDecision(ctx context.Context, mf *model.MediaFile, clientInfo *ClientInfo) (*Decision, error) {
	decision := &Decision{
		MediaID: mf.ID,
	}

	sourceBitrate := mf.BitRate // kbps

	log.Trace(ctx, "Making transcode decision", "mediaID", mf.ID, "container", mf.Suffix,
		"codec", mf.AudioCodec(), "bitrate", sourceBitrate, "channels", mf.Channels,
		"sampleRate", mf.SampleRate, "lossless", mf.IsLossless(), "client", clientInfo.Name)

	// Build source stream details
	decision.SourceStream = buildSourceStream(mf)

	// Check global bitrate constraint first.
	if clientInfo.MaxAudioBitrate > 0 && sourceBitrate > clientInfo.MaxAudioBitrate {
		log.Trace(ctx, "Global bitrate constraint exceeded, skipping direct play",
			"sourceBitrate", sourceBitrate, "maxAudioBitrate", clientInfo.MaxAudioBitrate)
		decision.TranscodeReasons = append(decision.TranscodeReasons, "audio bitrate not supported")
		// Skip direct play profiles entirely — global constraint fails
	} else {
		// Try direct play profiles, collecting reasons for each failure
		for _, profile := range clientInfo.DirectPlayProfiles {
			if reason := s.checkDirectPlayProfile(mf, sourceBitrate, &profile, clientInfo); reason == "" {
				decision.CanDirectPlay = true
				decision.TranscodeReasons = nil // Clear any previously collected reasons
				break
			} else {
				decision.TranscodeReasons = append(decision.TranscodeReasons, reason)
			}
		}
	}

	// If direct play is possible, we're done
	if decision.CanDirectPlay {
		log.Debug(ctx, "Transcode decision: direct play", "mediaID", mf.ID, "container", mf.Suffix, "codec", mf.AudioCodec())
		return decision, nil
	}

	// Try transcoding profiles (in order of preference)
	for _, profile := range clientInfo.TranscodingProfiles {
		if ts, transcodeFormat := s.computeTranscodedStream(ctx, mf, sourceBitrate, &profile, clientInfo); ts != nil {
			decision.CanTranscode = true
			decision.TargetFormat = transcodeFormat
			decision.TargetBitrate = ts.Bitrate
			decision.TargetChannels = ts.Channels
			decision.TargetSampleRate = ts.SampleRate
			decision.TargetBitDepth = ts.BitDepth
			decision.TranscodeStream = ts
			break
		}
	}

	if decision.CanTranscode {
		log.Debug(ctx, "Transcode decision: transcode", "mediaID", mf.ID,
			"targetFormat", decision.TargetFormat, "targetBitrate", decision.TargetBitrate,
			"targetChannels", decision.TargetChannels, "reasons", decision.TranscodeReasons)
	}

	// If neither direct play nor transcode is possible
	if !decision.CanDirectPlay && !decision.CanTranscode {
		decision.ErrorReason = "no compatible playback profile found"
		log.Warn(ctx, "Transcode decision: no compatible profile", "mediaID", mf.ID,
			"container", mf.Suffix, "codec", mf.AudioCodec(), "reasons", decision.TranscodeReasons)
	}

	return decision, nil
}

func buildSourceStream(mf *model.MediaFile) StreamDetails {
	return StreamDetails{
		Container:  mf.Suffix,
		Codec:      mf.AudioCodec(),
		Bitrate:    mf.BitRate,
		SampleRate: mf.SampleRate,
		BitDepth:   mf.BitDepth,
		Channels:   mf.Channels,
		Duration:   mf.Duration,
		Size:       mf.Size,
		IsLossless: mf.IsLossless(),
	}
}

// checkDirectPlayProfile returns "" if the profile matches (direct play OK),
// or a typed reason string if it doesn't match.
func (s *deciderService) checkDirectPlayProfile(mf *model.MediaFile, sourceBitrate int, profile *DirectPlayProfile, clientInfo *ClientInfo) string {
	// Check protocol (only http for now)
	if len(profile.Protocols) > 0 && !containsIgnoreCase(profile.Protocols, ProtocolHTTP) {
		return "protocol not supported"
	}

	// Check container
	if len(profile.Containers) > 0 && !matchesContainer(mf.Suffix, profile.Containers) {
		return "container not supported"
	}

	// Check codec
	if len(profile.AudioCodecs) > 0 && !matchesCodec(mf.AudioCodec(), profile.AudioCodecs) {
		return "audio codec not supported"
	}

	// Check channels
	if profile.MaxAudioChannels > 0 && mf.Channels > profile.MaxAudioChannels {
		return "audio channels not supported"
	}

	// Check codec-specific limitations
	for _, codecProfile := range clientInfo.CodecProfiles {
		if strings.EqualFold(codecProfile.Type, CodecProfileTypeAudio) && matchesCodec(mf.AudioCodec(), []string{codecProfile.Name}) {
			if reason := checkLimitations(mf, sourceBitrate, codecProfile.Limitations); reason != "" {
				return reason
			}
		}
	}

	return ""
}

// computeTranscodedStream attempts to build a valid transcoded stream for the given profile.
// Returns the stream details and the internal transcoding format (which may differ from the
// response container when a codec fallback occurs, e.g., "mp4"→"aac").
// Returns nil, "" if the profile cannot produce a valid output.
func (s *deciderService) computeTranscodedStream(ctx context.Context, mf *model.MediaFile, sourceBitrate int, profile *Profile, clientInfo *ClientInfo) (*StreamDetails, string) {
	// Check protocol (only http for now)
	if profile.Protocol != "" && !strings.EqualFold(profile.Protocol, ProtocolHTTP) {
		log.Trace(ctx, "Skipping transcoding profile: unsupported protocol", "protocol", profile.Protocol)
		return nil, ""
	}

	responseContainer, targetFormat := s.resolveTargetFormat(ctx, profile)
	if targetFormat == "" {
		return nil, ""
	}

	targetIsLossless := isLosslessFormat(targetFormat)

	// Reject lossy to lossless conversion
	if !mf.IsLossless() && targetIsLossless {
		log.Trace(ctx, "Skipping transcoding profile: lossy to lossless not allowed", "targetFormat", targetFormat)
		return nil, ""
	}

	ts := &StreamDetails{
		Container:  responseContainer,
		Codec:      strings.ToLower(profile.AudioCodec),
		SampleRate: normalizeSourceSampleRate(mf.SampleRate, mf.AudioCodec()),
		Channels:   mf.Channels,
		BitDepth:   normalizeSourceBitDepth(mf.BitDepth, mf.AudioCodec()),
		IsLossless: targetIsLossless,
	}
	if ts.Codec == "" {
		ts.Codec = targetFormat
	}

	// Apply codec-intrinsic sample rate adjustments before codec profile limitations
	if fixedRate := codecFixedOutputSampleRate(ts.Codec); fixedRate > 0 {
		ts.SampleRate = fixedRate
	}
	if maxRate := codecMaxSampleRate(ts.Codec); maxRate > 0 && ts.SampleRate > maxRate {
		ts.SampleRate = maxRate
	}

	// Determine target bitrate (all in kbps)
	if ok := s.computeBitrate(ctx, mf, sourceBitrate, targetFormat, targetIsLossless, clientInfo, ts); !ok {
		return nil, ""
	}

	// Apply MaxAudioChannels from the transcoding profile
	if profile.MaxAudioChannels > 0 && mf.Channels > profile.MaxAudioChannels {
		ts.Channels = profile.MaxAudioChannels
	}

	// Apply codec profile limitations to the TARGET codec
	if ok := s.applyCodecLimitations(ctx, sourceBitrate, targetFormat, targetIsLossless, clientInfo, ts); !ok {
		return nil, ""
	}

	return ts, targetFormat
}

// resolveTargetFormat determines the response container and internal target format
// by looking up transcoding configs. Returns ("", "") if no config found.
func (s *deciderService) resolveTargetFormat(ctx context.Context, profile *Profile) (responseContainer, targetFormat string) {
	responseContainer = strings.ToLower(profile.Container)
	targetFormat = responseContainer
	if targetFormat == "" {
		targetFormat = strings.ToLower(profile.AudioCodec)
		responseContainer = targetFormat
	}

	// Try the container first, then fall back to the audioCodec (e.g. "ogg" → "opus", "mp4" → "aac").
	_, err := s.ds.Transcoding(ctx).FindByFormat(targetFormat)
	if errors.Is(err, model.ErrNotFound) && profile.AudioCodec != "" && !strings.EqualFold(targetFormat, profile.AudioCodec) {
		codec := strings.ToLower(profile.AudioCodec)
		log.Trace(ctx, "No transcoding config for container, trying audioCodec", "container", targetFormat, "audioCodec", codec)
		_, err = s.ds.Transcoding(ctx).FindByFormat(codec)
		if err == nil {
			targetFormat = codec
		}
	}
	if err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			log.Error(ctx, "Error looking up transcoding config", "format", targetFormat, err)
		} else {
			log.Trace(ctx, "Skipping transcoding profile: no transcoding config", "targetFormat", targetFormat)
		}
		return "", ""
	}
	return responseContainer, targetFormat
}

// computeBitrate determines the target bitrate for the transcoded stream.
// Returns false if the profile should be rejected.
func (s *deciderService) computeBitrate(ctx context.Context, mf *model.MediaFile, sourceBitrate int, targetFormat string, targetIsLossless bool, clientInfo *ClientInfo, ts *StreamDetails) bool {
	if mf.IsLossless() {
		if !targetIsLossless {
			if clientInfo.MaxTranscodingAudioBitrate > 0 {
				ts.Bitrate = clientInfo.MaxTranscodingAudioBitrate
			} else {
				ts.Bitrate = defaultBitrate
			}
		} else {
			if clientInfo.MaxAudioBitrate > 0 && sourceBitrate > clientInfo.MaxAudioBitrate {
				log.Trace(ctx, "Skipping transcoding profile: lossless target exceeds bitrate limit",
					"targetFormat", targetFormat, "sourceBitrate", sourceBitrate, "maxAudioBitrate", clientInfo.MaxAudioBitrate)
				return false
			}
		}
	} else {
		ts.Bitrate = sourceBitrate
	}

	// Apply maxAudioBitrate as final cap
	if clientInfo.MaxAudioBitrate > 0 && ts.Bitrate > 0 && ts.Bitrate > clientInfo.MaxAudioBitrate {
		ts.Bitrate = clientInfo.MaxAudioBitrate
	}
	return true
}

// applyCodecLimitations applies codec profile limitations to the transcoded stream.
// Returns false if the profile should be rejected.
func (s *deciderService) applyCodecLimitations(ctx context.Context, sourceBitrate int, targetFormat string, targetIsLossless bool, clientInfo *ClientInfo, ts *StreamDetails) bool {
	targetCodec := ts.Codec
	for _, codecProfile := range clientInfo.CodecProfiles {
		if !strings.EqualFold(codecProfile.Type, CodecProfileTypeAudio) {
			continue
		}
		if !matchesCodec(targetCodec, []string{codecProfile.Name}) {
			continue
		}
		for _, lim := range codecProfile.Limitations {
			result := applyLimitation(sourceBitrate, &lim, ts)
			if strings.EqualFold(lim.Name, LimitationAudioBitrate) && targetIsLossless && result == adjustAdjusted {
				log.Trace(ctx, "Skipping transcoding profile: cannot adjust bitrate for lossless target",
					"targetFormat", targetFormat, "codec", targetCodec, "limitation", lim.Name)
				return false
			}
			if result == adjustCannotFit {
				log.Trace(ctx, "Skipping transcoding profile: codec limitation cannot be satisfied",
					"targetFormat", targetFormat, "codec", targetCodec, "limitation", lim.Name,
					"comparison", lim.Comparison, "values", lim.Values)
				return false
			}
		}
	}
	return true
}

func (s *deciderService) CreateTranscodeParams(decision *Decision) (string, error) {
	exp := time.Now().Add(tokenTTL)
	claims := map[string]any{
		"mid": decision.MediaID,
		"dp":  decision.CanDirectPlay,
	}
	if decision.CanTranscode && decision.TargetFormat != "" {
		claims["fmt"] = decision.TargetFormat
		claims["br"] = decision.TargetBitrate
		if decision.TargetChannels > 0 {
			claims["ch"] = decision.TargetChannels
		}
		if decision.TargetSampleRate > 0 {
			claims["sr"] = decision.TargetSampleRate
		}
		if decision.TargetBitDepth > 0 {
			claims["bd"] = decision.TargetBitDepth
		}
	}
	return auth.CreateExpiringPublicToken(exp, claims)
}

func (s *deciderService) ParseTranscodeParams(token string) (*Params, error) {
	claims, err := auth.Validate(token)
	if err != nil {
		return nil, err
	}

	params := &Params{}

	// Required claims
	mid, ok := claims["mid"].(string)
	if !ok || mid == "" {
		return nil, fmt.Errorf("invalid transcode token: missing media ID")
	}
	params.MediaID = mid

	dp, ok := claims["dp"].(bool)
	if !ok {
		return nil, fmt.Errorf("invalid transcode token: missing direct play flag")
	}
	params.DirectPlay = dp

	// Optional claims (legitimately absent for direct-play tokens)
	if f, ok := claims["fmt"].(string); ok {
		params.TargetFormat = f
	}
	if br, ok := claims["br"].(float64); ok {
		params.TargetBitrate = int(br)
	}
	if ch, ok := claims["ch"].(float64); ok {
		params.TargetChannels = int(ch)
	}
	if sr, ok := claims["sr"].(float64); ok {
		params.TargetSampleRate = int(sr)
	}
	if bd, ok := claims["bd"].(float64); ok {
		params.TargetBitDepth = int(bd)
	}

	return params, nil
}
