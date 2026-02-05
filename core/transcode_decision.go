package core

import (
	"context"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
)

const (
	transcodeTokenTTL       = 12 * time.Hour
	defaultTranscodeBitrate = 256 // kbps
)

// TranscodeDecision is the core service interface for making transcoding decisions
type TranscodeDecision interface {
	MakeDecision(ctx context.Context, mf *model.MediaFile, clientInfo *ClientInfo) (*Decision, error)
	CreateToken(decision *Decision) (string, error)
	ParseToken(token string) (*TranscodeParams, error)
}

// ClientInfo represents client playback capabilities
type ClientInfo struct {
	Name                       string
	Platform                   string
	MaxAudioBitrate            int
	MaxTranscodingAudioBitrate int
	DirectPlayProfiles         []DirectPlayProfile
	TranscodingProfiles        []TranscodingProfile
	CodecProfiles              []CodecProfile
}

// DirectPlayProfile describes a format the client can play directly
type DirectPlayProfile struct {
	Containers       []string
	AudioCodecs      []string
	Protocols        []string
	MaxAudioChannels int
}

// TranscodingProfile describes a transcoding target the client supports
type TranscodingProfile struct {
	Container        string
	AudioCodec       string
	Protocol         string
	MaxAudioChannels int
}

// CodecProfile describes codec-specific limitations
type CodecProfile struct {
	Type        string
	Name        string
	Limitations []Limitation
}

// Limitation describes a specific codec limitation
type Limitation struct {
	Property  string
	Condition string
	Value     string
}

// Decision represents the internal decision result
type Decision struct {
	MediaID          string
	CanDirectPlay    bool
	CanTranscode     bool
	TranscodeReasons []string
	ErrorReason      string
	TargetFormat     string
	TargetBitrate    int
	TargetChannels   int
	SourceStream     StreamDetails
	TranscodeStream  *StreamDetails
}

// StreamDetails describes audio stream properties
type StreamDetails struct {
	Container  string
	Codec      string
	Bitrate    int
	SampleRate int
	BitDepth   int
	Channels   int
	Duration   float32
	Size       int64
	IsLossless bool
}

// TranscodeParams contains the parameters extracted from a transcode token
type TranscodeParams struct {
	MediaID        string
	DirectPlay     bool
	TargetFormat   string
	TargetBitrate  int
	TargetChannels int
}

func NewTranscodeDecision(ds model.DataStore) TranscodeDecision {
	return &transcodeDecisionService{
		ds: ds,
	}
}

type transcodeDecisionService struct {
	ds model.DataStore
}

func (s *transcodeDecisionService) MakeDecision(ctx context.Context, mf *model.MediaFile, clientInfo *ClientInfo) (*Decision, error) {
	decision := &Decision{
		MediaID: mf.ID,
	}

	// Build source stream details
	decision.SourceStream = StreamDetails{
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

	// Check global bitrate constraint
	if clientInfo.MaxAudioBitrate > 0 && mf.BitRate > clientInfo.MaxAudioBitrate {
		decision.TranscodeReasons = append(decision.TranscodeReasons, "bitrate exceeds maxAudioBitrate")
	}

	// Try direct play profiles
	for _, profile := range clientInfo.DirectPlayProfiles {
		if s.matchesDirectPlayProfile(mf, &profile, clientInfo) {
			decision.CanDirectPlay = true
			break
		}
	}

	// If direct play is possible and no transcode reasons, we're done
	if decision.CanDirectPlay && len(decision.TranscodeReasons) == 0 {
		return decision, nil
	}

	// Try transcoding profiles (in order of preference)
	for _, profile := range clientInfo.TranscodingProfiles {
		if targetFormat, targetBitrate, ok := s.matchesTranscodingProfile(ctx, mf, &profile, clientInfo); ok {
			decision.CanTranscode = true
			decision.TargetFormat = targetFormat
			decision.TargetBitrate = targetBitrate
			decision.TargetChannels = profile.MaxAudioChannels

			// Build transcode stream details
			decision.TranscodeStream = &StreamDetails{
				Container:  targetFormat,
				Codec:      targetFormat,
				Bitrate:    targetBitrate,
				SampleRate: mf.SampleRate,
				Channels:   mf.Channels,
				IsLossless: false,
			}
			if decision.TargetChannels > 0 && decision.TargetChannels < mf.Channels {
				decision.TranscodeStream.Channels = decision.TargetChannels
			}
			break
		}
	}

	// If neither direct play nor transcode is possible
	if !decision.CanDirectPlay && !decision.CanTranscode {
		decision.ErrorReason = "no compatible playback profile found"
	}

	return decision, nil
}

func (s *transcodeDecisionService) matchesDirectPlayProfile(mf *model.MediaFile, profile *DirectPlayProfile, clientInfo *ClientInfo) bool {
	// Check protocol (only http for now)
	if len(profile.Protocols) > 0 && !containsIgnoreCase(profile.Protocols, "http") {
		return false
	}

	// Check container
	if len(profile.Containers) > 0 && !s.matchesContainer(mf.Suffix, profile.Containers) {
		return false
	}

	// Check codec
	if len(profile.AudioCodecs) > 0 && !s.matchesCodec(mf.AudioCodec(), profile.AudioCodecs) {
		return false
	}

	// Check channels
	if profile.MaxAudioChannels > 0 && mf.Channels > profile.MaxAudioChannels {
		return false
	}

	// Check codec-specific limitations
	for _, codecProfile := range clientInfo.CodecProfiles {
		if strings.EqualFold(codecProfile.Type, "AudioCodec") && strings.EqualFold(codecProfile.Name, mf.AudioCodec()) {
			if !s.meetsLimitations(mf, codecProfile.Limitations) {
				return false
			}
		}
	}

	return true
}

func (s *transcodeDecisionService) matchesTranscodingProfile(ctx context.Context, mf *model.MediaFile, profile *TranscodingProfile, clientInfo *ClientInfo) (string, int, bool) {
	// Check protocol (only http for now)
	if profile.Protocol != "" && !strings.EqualFold(profile.Protocol, "http") {
		return "", 0, false
	}

	targetFormat := strings.ToLower(profile.Container)
	if targetFormat == "" {
		targetFormat = strings.ToLower(profile.AudioCodec)
	}

	// Verify we have a transcoding config for this format
	tc, err := s.ds.Transcoding(ctx).FindByFormat(targetFormat)
	if err != nil || tc == nil {
		return "", 0, false
	}

	// Reject lossy to lossless conversion
	if !mf.IsLossless() && isLosslessFormat(targetFormat) {
		return "", 0, false
	}

	// Determine target bitrate
	targetBitrate := defaultTranscodeBitrate
	if mf.IsLossless() {
		// Lossless to lossy: use client's max transcoding bitrate or default
		if clientInfo.MaxTranscodingAudioBitrate > 0 {
			targetBitrate = clientInfo.MaxTranscodingAudioBitrate / 1000 // Convert to kbps
		}
	} else {
		// Lossy to lossy: try to preserve source bitrate if under max
		targetBitrate = mf.BitRate / 1000
		if clientInfo.MaxTranscodingAudioBitrate > 0 && targetBitrate > clientInfo.MaxTranscodingAudioBitrate/1000 {
			targetBitrate = clientInfo.MaxTranscodingAudioBitrate / 1000
		}
	}

	return targetFormat, targetBitrate, true
}

func (s *transcodeDecisionService) matchesContainer(suffix string, containers []string) bool {
	suffix = strings.ToLower(suffix)
	for _, c := range containers {
		c = strings.ToLower(c)
		if c == suffix {
			return true
		}
		// Handle common aliases
		if c == "aac" && (suffix == "m4a" || suffix == "m4b" || suffix == "m4p") {
			return true
		}
		if c == "mpeg" && (suffix == "mp3" || suffix == "mp2") {
			return true
		}
		if c == "ogg" && (suffix == "oga" || suffix == "opus") {
			return true
		}
	}
	return false
}

func (s *transcodeDecisionService) matchesCodec(codec string, codecs []string) bool {
	codec = strings.ToLower(codec)
	for _, c := range codecs {
		c = strings.ToLower(c)
		if c == codec {
			return true
		}
		// Handle common aliases
		if c == "aac" && codec == "alac" {
			continue // ALAC is not AAC
		}
	}
	return false
}

func (s *transcodeDecisionService) meetsLimitations(mf *model.MediaFile, limitations []Limitation) bool {
	for _, lim := range limitations {
		switch strings.ToLower(lim.Property) {
		case "audiochannels":
			if !checkIntLimitation(mf.Channels, lim.Condition, lim.Value) {
				return false
			}
		case "audiosamplerate":
			if !checkIntLimitation(mf.SampleRate, lim.Condition, lim.Value) {
				return false
			}
		case "audiobitrate":
			if !checkIntLimitation(mf.BitRate, lim.Condition, lim.Value) {
				return false
			}
		case "audiobitdepth":
			if !checkIntLimitation(mf.BitDepth, lim.Condition, lim.Value) {
				return false
			}
		}
	}
	return true
}

func (s *transcodeDecisionService) CreateToken(decision *Decision) (string, error) {
	exp := time.Now().Add(transcodeTokenTTL)
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
	}
	return auth.CreateExpiringPublicToken(exp, claims)
}

func (s *transcodeDecisionService) ParseToken(token string) (*TranscodeParams, error) {
	claims, err := auth.Validate(token)
	if err != nil {
		return nil, err
	}

	params := &TranscodeParams{}
	if mid, ok := claims["mid"].(string); ok {
		params.MediaID = mid
	}
	if dp, ok := claims["dp"].(bool); ok {
		params.DirectPlay = dp
	}
	if fmt, ok := claims["fmt"].(string); ok {
		params.TargetFormat = fmt
	}
	if br, ok := claims["br"].(float64); ok {
		params.TargetBitrate = int(br)
	}
	if ch, ok := claims["ch"].(float64); ok {
		params.TargetChannels = int(ch)
	}

	return params, nil
}

func containsIgnoreCase(slice []string, s string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, s) {
			return true
		}
	}
	return false
}

func checkIntLimitation(value int, condition, limitValue string) bool {
	var limit int
	if _, err := parseIntFromString(limitValue, &limit); err != nil {
		return true // If we can't parse the limit, assume it passes
	}

	switch strings.ToLower(condition) {
	case "lessthanequal", "lte":
		return value <= limit
	case "greaterthanequal", "gte":
		return value >= limit
	case "equals", "eq":
		return value == limit
	case "notequals", "ne":
		return value != limit
	default:
		return true
	}
}

func parseIntFromString(s string, out *int) (bool, error) {
	var v int
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		v = v*10 + int(c-'0')
	}
	*out = v
	return true, nil
}

func isLosslessFormat(format string) bool {
	switch strings.ToLower(format) {
	case "flac", "alac", "wav", "aiff", "ape", "wv", "tta", "tak", "shn", "dsd":
		return true
	}
	return false
}
