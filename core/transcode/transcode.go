package transcode

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"encoding/json"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const (
	tokenTTL       = 12 * time.Hour
	defaultBitrate = 256 // kbps
)

func NewDecider(ds model.DataStore, ff ffmpeg.FFmpeg) Decider {
	return &deciderService{
		ds: ds,
		ff: ff,
	}
}

type deciderService struct {
	ds model.DataStore
	ff ffmpeg.FFmpeg
}

func (s *deciderService) MakeDecision(ctx context.Context, mf *model.MediaFile, clientInfo *ClientInfo) (*Decision, error) {
	decision := &Decision{
		MediaID:         mf.ID,
		SourceUpdatedAt: mf.UpdatedAt,
	}

	probe, err := s.ensureProbed(ctx, mf)
	if err != nil {
		return nil, err
	}

	// Build source stream details (uses probe data if available)
	decision.SourceStream = buildSourceStream(mf, probe)
	src := &decision.SourceStream

	log.Trace(ctx, "Making transcode decision", "mediaID", mf.ID, "container", src.Container,
		"codec", src.Codec, "bitrate", src.Bitrate, "channels", src.Channels,
		"sampleRate", src.SampleRate, "lossless", src.IsLossless, "client", clientInfo.Name)

	// Check global bitrate constraint first.
	if clientInfo.MaxAudioBitrate > 0 && src.Bitrate > clientInfo.MaxAudioBitrate {
		log.Trace(ctx, "Global bitrate constraint exceeded, skipping direct play",
			"sourceBitrate", src.Bitrate, "maxAudioBitrate", clientInfo.MaxAudioBitrate)
		decision.TranscodeReasons = append(decision.TranscodeReasons, "audio bitrate not supported")
		// Skip direct play profiles entirely — global constraint fails
	} else {
		// Try direct play profiles, collecting reasons for each failure
		for _, profile := range clientInfo.DirectPlayProfiles {
			if reason := s.checkDirectPlayProfile(src, &profile, clientInfo); reason == "" {
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
		log.Debug(ctx, "Transcode decision: direct play", "mediaID", mf.ID, "container", src.Container, "codec", src.Codec)
		return decision, nil
	}

	// Try transcoding profiles (in order of preference)
	for _, profile := range clientInfo.TranscodingProfiles {
		if ts, transcodeFormat := s.computeTranscodedStream(ctx, src, &profile, clientInfo); ts != nil {
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
			"container", src.Container, "codec", src.Codec, "reasons", decision.TranscodeReasons)
	}

	return decision, nil
}

func buildSourceStream(mf *model.MediaFile, probe *ffmpeg.AudioProbeResult) StreamDetails {
	sd := StreamDetails{
		Container: mf.Suffix,
		Duration:  mf.Duration,
		Size:      mf.Size,
	}

	// Use pre-parsed probe result, or fall back to parsing stored probe data
	if probe == nil {
		probe, _ = parseProbeData(mf.ProbeData)
	}

	// Use probe data if available for authoritative values
	if probe != nil {
		sd.Codec = normalizeProbeCodec(probe.Codec)
		sd.Profile = probe.Profile
		sd.Bitrate = probe.BitRate
		sd.SampleRate = probe.SampleRate
		sd.BitDepth = probe.BitDepth
		sd.Channels = probe.Channels
	} else {
		sd.Codec = mf.AudioCodec()
		sd.Bitrate = mf.BitRate
		sd.SampleRate = mf.SampleRate
		sd.BitDepth = mf.BitDepth
		sd.Channels = mf.Channels
	}

	sd.IsLossless = mf.IsLossless()
	return sd
}

func parseProbeData(data string) (*ffmpeg.AudioProbeResult, error) {
	if data == "" {
		return nil, nil
	}
	var result ffmpeg.AudioProbeResult
	err := json.Unmarshal([]byte(data), &result)
	return &result, err
}

// checkDirectPlayProfile returns "" if the profile matches (direct play OK),
// or a typed reason string if it doesn't match.
func (s *deciderService) checkDirectPlayProfile(src *StreamDetails, profile *DirectPlayProfile, clientInfo *ClientInfo) string {
	// Check protocol (only http for now)
	if len(profile.Protocols) > 0 && !containsIgnoreCase(profile.Protocols, ProtocolHTTP) {
		return "protocol not supported"
	}

	// Check container
	if len(profile.Containers) > 0 && !matchesContainer(src.Container, profile.Containers) {
		return "container not supported"
	}

	// Check codec
	if len(profile.AudioCodecs) > 0 && !matchesCodec(src.Codec, profile.AudioCodecs) {
		return "audio codec not supported"
	}

	// Check channels
	if profile.MaxAudioChannels > 0 && src.Channels > profile.MaxAudioChannels {
		return "audio channels not supported"
	}

	// Check codec-specific limitations
	for _, codecProfile := range clientInfo.CodecProfiles {
		if strings.EqualFold(codecProfile.Type, CodecProfileTypeAudio) && matchesCodec(src.Codec, []string{codecProfile.Name}) {
			if reason := checkLimitations(src, codecProfile.Limitations); reason != "" {
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
func (s *deciderService) computeTranscodedStream(ctx context.Context, src *StreamDetails, profile *Profile, clientInfo *ClientInfo) (*StreamDetails, string) {
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
	if !src.IsLossless && targetIsLossless {
		log.Trace(ctx, "Skipping transcoding profile: lossy to lossless not allowed", "targetFormat", targetFormat)
		return nil, ""
	}

	ts := &StreamDetails{
		Container:  responseContainer,
		Codec:      strings.ToLower(profile.AudioCodec),
		SampleRate: normalizeSourceSampleRate(src.SampleRate, src.Codec),
		Channels:   src.Channels,
		BitDepth:   normalizeSourceBitDepth(src.BitDepth, src.Codec),
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
	if ok := s.computeBitrate(ctx, src, targetFormat, targetIsLossless, clientInfo, ts); !ok {
		return nil, ""
	}

	// Apply MaxAudioChannels from the transcoding profile
	if profile.MaxAudioChannels > 0 && src.Channels > profile.MaxAudioChannels {
		ts.Channels = profile.MaxAudioChannels
	}

	// Apply codec profile limitations to the TARGET codec
	if ok := s.applyCodecLimitations(ctx, src.Bitrate, targetFormat, targetIsLossless, clientInfo, ts); !ok {
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
func (s *deciderService) computeBitrate(ctx context.Context, src *StreamDetails, targetFormat string, targetIsLossless bool, clientInfo *ClientInfo, ts *StreamDetails) bool {
	if src.IsLossless {
		if !targetIsLossless {
			if clientInfo.MaxTranscodingAudioBitrate > 0 {
				ts.Bitrate = clientInfo.MaxTranscodingAudioBitrate
			} else {
				ts.Bitrate = defaultBitrate
			}
		} else {
			if clientInfo.MaxAudioBitrate > 0 && src.Bitrate > clientInfo.MaxAudioBitrate {
				log.Trace(ctx, "Skipping transcoding profile: lossless target exceeds bitrate limit",
					"targetFormat", targetFormat, "sourceBitrate", src.Bitrate, "maxAudioBitrate", clientInfo.MaxAudioBitrate)
				return false
			}
		}
	} else {
		ts.Bitrate = src.Bitrate
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

// ensureProbed runs ffprobe if probe data is missing, persists it, and returns
// the parsed result. Returns (nil, nil) when probing is skipped or data already exists
// (in which case the caller should parse mf.ProbeData).
func (s *deciderService) ensureProbed(ctx context.Context, mf *model.MediaFile) (*ffmpeg.AudioProbeResult, error) {
	if mf.ProbeData != "" {
		return nil, nil
	}
	if !conf.Server.DevEnableMediaFileProbe {
		return nil, nil
	}

	result, err := s.ff.ProbeAudioStream(ctx, mf.AbsolutePath())
	if err != nil {
		return nil, fmt.Errorf("probing media file %s: %w", mf.ID, err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling probe result for %s: %w", mf.ID, err)
	}
	mf.ProbeData = string(data)

	if err := s.ds.MediaFile(ctx).UpdateProbeData(mf.ID, mf.ProbeData); err != nil {
		log.Error(ctx, "Failed to persist probe data", "mediaID", mf.ID, err)
		// Don't fail the decision — we have the data in memory
	}

	log.Debug(ctx, "Probed media file", "mediaID", mf.ID, "codec", result.Codec,
		"profile", result.Profile, "bitRate", result.BitRate,
		"sampleRate", result.SampleRate, "bitDepth", result.BitDepth, "channels", result.Channels)
	return result, nil
}

func (s *deciderService) CreateTranscodeParams(decision *Decision) (string, error) {
	return auth.EncodeToken(decision.toClaimsMap())
}

func (s *deciderService) ParseTranscodeParams(tokenStr string) (*Params, error) {
	token, err := auth.DecodeAndVerifyToken(tokenStr)
	if err != nil {
		return nil, err
	}
	return paramsFromToken(token)
}

func (s *deciderService) ValidateTranscodeParams(ctx context.Context, token string, mediaID string) (*Params, *model.MediaFile, error) {
	params, err := s.ParseTranscodeParams(token)
	if err != nil {
		return nil, nil, errors.Join(ErrTokenInvalid, err)
	}
	if params.MediaID != mediaID {
		return nil, nil, fmt.Errorf("%w: token mediaID %q does not match %q", ErrTokenInvalid, params.MediaID, mediaID)
	}
	mf, err := s.ds.MediaFile(ctx).Get(mediaID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, nil, ErrMediaNotFound
		}
		return nil, nil, err
	}
	if !mf.UpdatedAt.Truncate(time.Second).Equal(params.SourceUpdatedAt) {
		log.Info(ctx, "Transcode token is stale", "mediaID", mediaID,
			"tokenUpdatedAt", params.SourceUpdatedAt, "fileUpdatedAt", mf.UpdatedAt)
		return nil, nil, ErrTokenStale
	}
	return params, mf, nil
}
