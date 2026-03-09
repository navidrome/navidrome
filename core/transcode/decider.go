package transcode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

const defaultBitrate = 256 // kbps

// Decider is the core service interface for making transcoding decisions
type Decider interface {
	MakeDecision(ctx context.Context, mf *model.MediaFile, clientInfo *ClientInfo, opts DecisionOptions) (*Decision, error)
	CreateTranscodeParams(decision *Decision) (string, error)
	ResolveRequestFromToken(ctx context.Context, token string, mediaID string, offset int) (StreamRequest, *model.MediaFile, error)
	ResolveRequest(ctx context.Context, mf *model.MediaFile, reqFormat string, reqBitRate int, offset int) StreamRequest
}

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

func (s *deciderService) MakeDecision(ctx context.Context, mf *model.MediaFile, clientInfo *ClientInfo, opts DecisionOptions) (*Decision, error) {
	decision := &Decision{
		MediaID:         mf.ID,
		SourceUpdatedAt: mf.UpdatedAt,
	}

	var probe *ffmpeg.AudioProbeResult
	if !opts.SkipProbe {
		var err error
		probe, err = s.ensureProbed(ctx, mf)
		if err != nil {
			return nil, err
		}
	}

	// Build source stream details (uses probe data if available)
	decision.SourceStream = buildSourceStream(mf, probe)
	src := &decision.SourceStream

	// Check for server-side player transcoding override
	if trc, ok := request.TranscodingFrom(ctx); ok && trc.TargetFormat != "" {
		clientInfo = applyServerOverride(ctx, clientInfo, &trc)
	}

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
	sd.IsLossless = isLosslessFormat(sd.Codec)

	return sd
}

// applyServerOverride replaces the client-provided profiles with synthetic ones
// matching the server-forced transcoding format and bitrate.
func applyServerOverride(ctx context.Context, original *ClientInfo, trc *model.Transcoding) *ClientInfo {
	maxBitRate := trc.DefaultBitRate
	if player, ok := request.PlayerFrom(ctx); ok && player.MaxBitRate > 0 {
		maxBitRate = player.MaxBitRate
	}

	log.Debug(ctx, "Applying server-side transcoding override",
		"targetFormat", trc.TargetFormat, "maxBitRate", maxBitRate,
		"client", original.Name)

	return &ClientInfo{
		Name:                       original.Name,
		Platform:                   original.Platform,
		MaxAudioBitrate:            maxBitRate,
		MaxTranscodingAudioBitrate: maxBitRate,
		DirectPlayProfiles: []DirectPlayProfile{
			{Containers: []string{trc.TargetFormat}, AudioCodecs: []string{trc.TargetFormat}, Protocols: []string{ProtocolHTTP}},
		},
		TranscodingProfiles: []Profile{
			{Container: trc.TargetFormat, AudioCodec: trc.TargetFormat, Protocol: ProtocolHTTP},
		},
	}
}

func parseProbeData(data string) (*ffmpeg.AudioProbeResult, error) {
	if data == "" {
		return nil, nil
	}
	var result ffmpeg.AudioProbeResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}
	return &result, nil
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

	responseContainer, targetFormat := resolveTargetFormat(profile)
	if targetFormat == "" {
		return nil, ""
	}

	// Verify we have a transcoding command available (DB custom or built-in default)
	if LookupTranscodeCommand(ctx, s.ds, targetFormat) == "" {
		log.Trace(ctx, "Skipping transcoding profile: no transcoding command available", "targetFormat", targetFormat)
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

// LookupTranscodeCommand returns the ffmpeg command for the given format.
// It checks the DB first (for user-customized commands), then falls back to
// the built-in default command. Returns "" if the format is unknown.
func LookupTranscodeCommand(ctx context.Context, ds model.DataStore, format string) string {
	t, err := ds.Transcoding(ctx).FindByFormat(format)
	if err == nil && t.Command != "" {
		return t.Command
	}
	// Fall back to built-in defaults
	for _, dt := range consts.DefaultTranscodings {
		if dt.TargetFormat == format {
			return dt.Command
		}
	}
	return ""
}

// resolveTargetFormat determines the response container and internal target format
// from the profile's Container and AudioCodec fields. When an AudioCodec is specified
// it is preferred as targetFormat (e.g. container "mp4" with audioCodec "aac" → targetFormat "aac").
func resolveTargetFormat(profile *Profile) (responseContainer, targetFormat string) {
	responseContainer = strings.ToLower(profile.Container)
	targetFormat = responseContainer

	// Prefer the audioCodec as targetFormat when provided (handles container-to-codec
	// mapping like "mp4" → "aac", "ogg" → "opus").
	if profile.AudioCodec != "" {
		targetFormat = strings.ToLower(profile.AudioCodec)
	}

	// If neither container nor audioCodec is set, we can't resolve a format.
	if targetFormat == "" {
		return "", ""
	}

	// When no container was specified, use the targetFormat as container too.
	if responseContainer == "" {
		responseContainer = targetFormat
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
