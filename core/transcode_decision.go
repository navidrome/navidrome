package core

import (
	"context"
	"slices"
	"strconv"
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

// ClientInfo represents client playback capabilities.
// All bitrate values are in kilobits per second (kbps), matching Navidrome conventions.
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
	Name       string
	Comparison string
	Values     []string
	Required   bool
}

// Protocol values (OpenSubsonic spec enum)
const (
	ProtocolHTTP = "http"
	ProtocolHLS  = "hls"
)

// Comparison operators (OpenSubsonic spec enum)
const (
	ComparisonEquals           = "Equals"
	ComparisonNotEquals        = "NotEquals"
	ComparisonLessThanEqual    = "LessThanEqual"
	ComparisonGreaterThanEqual = "GreaterThanEqual"
)

// Limitation names (OpenSubsonic spec enum)
const (
	LimitationAudioChannels   = "audioChannels"
	LimitationAudioBitrate    = "audioBitrate"
	LimitationAudioProfile    = "audioProfile"
	LimitationAudioSamplerate = "audioSamplerate"
	LimitationAudioBitdepth   = "audioBitdepth"
)

// Codec profile types (OpenSubsonic spec enum)
const (
	CodecProfileTypeAudio = "AudioCodec"
)

// Decision represents the internal decision result.
// All bitrate values are in kilobits per second (kbps).
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

// StreamDetails describes audio stream properties.
// Bitrate is in kilobits per second (kbps).
type StreamDetails struct {
	Container  string
	Codec      string
	Profile    string // Audio profile (e.g., "LC", "HE-AAC"). Empty until scanner support is added.
	Bitrate    int
	SampleRate int
	BitDepth   int
	Channels   int
	Duration   float32
	Size       int64
	IsLossless bool
}

// TranscodeParams contains the parameters extracted from a transcode token.
// TargetBitrate is in kilobits per second (kbps).
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

	sourceBitrate := mf.BitRate // kbps

	// Build source stream details
	decision.SourceStream = StreamDetails{
		Container:  mf.Suffix,
		Codec:      mf.AudioCodec(),
		Bitrate:    sourceBitrate,
		SampleRate: mf.SampleRate,
		BitDepth:   mf.BitDepth,
		Channels:   mf.Channels,
		Duration:   mf.Duration,
		Size:       mf.Size,
		IsLossless: mf.IsLossless(),
	}

	// Check global bitrate constraint first.
	if clientInfo.MaxAudioBitrate > 0 && sourceBitrate > clientInfo.MaxAudioBitrate {
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
		return decision, nil
	}

	// Try transcoding profiles (in order of preference)
	for _, profile := range clientInfo.TranscodingProfiles {
		if ts := s.computeTranscodedStream(ctx, mf, sourceBitrate, &profile, clientInfo); ts != nil {
			decision.CanTranscode = true
			decision.TargetFormat = ts.Container
			decision.TargetBitrate = ts.Bitrate
			decision.TargetChannels = ts.Channels
			decision.TranscodeStream = ts
			break
		}
	}

	// If neither direct play nor transcode is possible
	if !decision.CanDirectPlay && !decision.CanTranscode {
		decision.ErrorReason = "no compatible playback profile found"
	}

	return decision, nil
}

// checkDirectPlayProfile returns "" if the profile matches (direct play OK),
// or a typed reason string if it doesn't match.
func (s *transcodeDecisionService) checkDirectPlayProfile(mf *model.MediaFile, sourceBitrate int, profile *DirectPlayProfile, clientInfo *ClientInfo) string {
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

// checkLimitations checks codec profile limitations against source media.
// Returns "" if all limitations pass, or a typed reason string for the first failure.
func checkLimitations(mf *model.MediaFile, sourceBitrate int, limitations []Limitation) string {
	for _, lim := range limitations {
		var ok bool
		var reason string

		switch lim.Name {
		case LimitationAudioChannels:
			ok = checkIntLimitation(mf.Channels, lim.Comparison, lim.Values)
			reason = "audio channels not supported"
		case LimitationAudioSamplerate:
			ok = checkIntLimitation(mf.SampleRate, lim.Comparison, lim.Values)
			reason = "audio samplerate not supported"
		case LimitationAudioBitrate:
			ok = checkIntLimitation(sourceBitrate, lim.Comparison, lim.Values)
			reason = "audio bitrate not supported"
		case LimitationAudioBitdepth:
			ok = checkIntLimitation(mf.BitDepth, lim.Comparison, lim.Values)
			reason = "audio bitdepth not supported"
		case LimitationAudioProfile:
			// TODO: populate source profile when MediaFile has audio profile info
			ok = checkStringLimitation("", lim.Comparison, lim.Values)
			reason = "audio profile not supported"
		default:
			continue
		}

		if !ok && lim.Required {
			return reason
		}
	}
	return ""
}

// adjustResult represents the outcome of applying a limitation to a transcoded stream value
type adjustResult int

const (
	adjustNone      adjustResult = iota // Value already satisfies the limitation
	adjustAdjusted                      // Value was changed to fit the limitation
	adjustCannotFit                     // Cannot satisfy the limitation (reject this profile)
)

// computeTranscodedStream attempts to build a valid transcoded stream for the given profile.
// Returns nil if the profile cannot produce a valid output.
func (s *transcodeDecisionService) computeTranscodedStream(ctx context.Context, mf *model.MediaFile, sourceBitrate int, profile *TranscodingProfile, clientInfo *ClientInfo) *StreamDetails {
	// Check protocol (only http for now)
	if profile.Protocol != "" && !strings.EqualFold(profile.Protocol, ProtocolHTTP) {
		return nil
	}

	targetFormat := strings.ToLower(profile.Container)
	if targetFormat == "" {
		targetFormat = strings.ToLower(profile.AudioCodec)
	}

	// Verify we have a transcoding config for this format
	tc, err := s.ds.Transcoding(ctx).FindByFormat(targetFormat)
	if err != nil || tc == nil {
		return nil
	}

	targetIsLossless := isLosslessFormat(targetFormat)

	// Reject lossy to lossless conversion
	if !mf.IsLossless() && targetIsLossless {
		return nil
	}

	ts := &StreamDetails{
		Container:  targetFormat,
		Codec:      strings.ToLower(profile.AudioCodec),
		SampleRate: mf.SampleRate,
		Channels:   mf.Channels,
		IsLossless: targetIsLossless,
	}
	if ts.Codec == "" {
		ts.Codec = targetFormat
	}

	// Determine target bitrate (all in kbps)
	if mf.IsLossless() {
		if !targetIsLossless {
			// Lossless to lossy: use client's max transcoding bitrate or default
			if clientInfo.MaxTranscodingAudioBitrate > 0 {
				ts.Bitrate = clientInfo.MaxTranscodingAudioBitrate
			} else {
				ts.Bitrate = defaultTranscodeBitrate
			}
		} else {
			// Lossless to lossless: check if bitrate is under the global max
			if clientInfo.MaxAudioBitrate > 0 && sourceBitrate > clientInfo.MaxAudioBitrate {
				return nil // Cannot guarantee bitrate within limit for lossless
			}
			// No explicit bitrate for lossless target (leave 0)
		}
	} else {
		// Lossy to lossy: preserve source bitrate
		ts.Bitrate = sourceBitrate
	}

	// Apply maxAudioBitrate as final cap on transcoded stream (#5)
	if clientInfo.MaxAudioBitrate > 0 && ts.Bitrate > 0 && ts.Bitrate > clientInfo.MaxAudioBitrate {
		ts.Bitrate = clientInfo.MaxAudioBitrate
	}

	// Apply MaxAudioChannels from the transcoding profile
	if profile.MaxAudioChannels > 0 && mf.Channels > profile.MaxAudioChannels {
		ts.Channels = profile.MaxAudioChannels
	}

	// Apply codec profile limitations to the TARGET codec (#4)
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
			// For lossless codecs, adjusting bitrate is not valid
			if strings.EqualFold(lim.Name, LimitationAudioBitrate) && targetIsLossless && result == adjustAdjusted {
				return nil
			}
			if result == adjustCannotFit {
				return nil
			}
		}
	}

	return ts
}

// applyLimitation adjusts a transcoded stream parameter to satisfy the limitation.
// Returns the adjustment result.
func applyLimitation(sourceBitrate int, lim *Limitation, ts *StreamDetails) adjustResult {
	switch lim.Name {
	case LimitationAudioChannels:
		return applyIntLimitation(lim.Comparison, lim.Values, ts.Channels, func(v int) { ts.Channels = v })
	case LimitationAudioBitrate:
		current := ts.Bitrate
		if current == 0 {
			current = sourceBitrate
		}
		return applyIntLimitation(lim.Comparison, lim.Values, current, func(v int) { ts.Bitrate = v })
	case LimitationAudioSamplerate:
		return applyIntLimitation(lim.Comparison, lim.Values, ts.SampleRate, func(v int) { ts.SampleRate = v })
	case LimitationAudioBitdepth:
		if ts.BitDepth > 0 {
			return applyIntLimitation(lim.Comparison, lim.Values, ts.BitDepth, func(v int) { ts.BitDepth = v })
		}
	case LimitationAudioProfile:
		// TODO: implement when audio profile data is available
	}
	return adjustNone
}

// applyIntLimitation applies a limitation comparison to a value.
// If the value needs adjusting, calls the setter and returns the result.
func applyIntLimitation(comparison string, values []string, current int, setter func(int)) adjustResult {
	if len(values) == 0 {
		return adjustNone
	}

	switch comparison {
	case ComparisonLessThanEqual:
		limit, ok := parseInt(values[0])
		if !ok {
			return adjustNone
		}
		if current <= limit {
			return adjustNone
		}
		setter(limit)
		return adjustAdjusted
	case ComparisonGreaterThanEqual:
		limit, ok := parseInt(values[0])
		if !ok {
			return adjustNone
		}
		if current >= limit {
			return adjustNone
		}
		// Cannot upscale
		return adjustCannotFit
	case ComparisonEquals:
		// Check if current value matches any allowed value
		for _, v := range values {
			if limit, ok := parseInt(v); ok && current == limit {
				return adjustNone
			}
		}
		// Find the closest allowed value below current (don't upscale)
		var closest int
		found := false
		for _, v := range values {
			if limit, ok := parseInt(v); ok && limit < current {
				if !found || limit > closest {
					closest = limit
					found = true
				}
			}
		}
		if found {
			setter(closest)
			return adjustAdjusted
		}
		return adjustCannotFit
	case ComparisonNotEquals:
		for _, v := range values {
			if limit, ok := parseInt(v); ok && current == limit {
				return adjustCannotFit
			}
		}
		return adjustNone
	}

	return adjustNone
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
	return slices.ContainsFunc(slice, func(item string) bool {
		return strings.EqualFold(item, s)
	})
}

// containerAliasGroups maps each container alias to a canonical group name.
var containerAliasGroups = func() map[string]string {
	groups := [][]string{
		{"aac", "adts", "m4a", "mp4", "m4b", "m4p"},
		{"mpeg", "mp3", "mp2"},
		{"ogg", "oga"},
		{"aif", "aiff"},
		{"asf", "wma"},
		{"mpc", "mpp"},
		{"wv"},
	}
	m := make(map[string]string)
	for _, g := range groups {
		canonical := g[0]
		for _, name := range g {
			m[name] = canonical
		}
	}
	return m
}()

// matchesWithAliases checks if a value matches any entry in candidates,
// consulting the alias map for equivalent names.
func matchesWithAliases(value string, candidates []string, aliases map[string]string) bool {
	value = strings.ToLower(value)
	canonical := aliases[value]
	for _, c := range candidates {
		c = strings.ToLower(c)
		if c == value {
			return true
		}
		if canonical != "" && aliases[c] == canonical {
			return true
		}
	}
	return false
}

// matchesContainer checks if a file suffix matches any of the container names,
// including common aliases.
func matchesContainer(suffix string, containers []string) bool {
	return matchesWithAliases(suffix, containers, containerAliasGroups)
}

// codecAliasGroups maps each codec alias to a canonical group name.
// Codecs within the same group are considered equivalent.
var codecAliasGroups = func() map[string]string {
	groups := [][]string{
		{"aac", "adts"},
		{"ac3", "ac-3"},
		{"eac3", "e-ac3", "e-ac-3", "eac-3"},
		{"mpc7", "musepack7"},
		{"mpc8", "musepack8"},
		{"wma1", "wmav1"},
		{"wma2", "wmav2"},
		{"wmalossless", "wma9lossless"},
		{"wmapro", "wma9pro"},
		{"shn", "shorten"},
		{"mp4als", "als"},
	}
	m := make(map[string]string)
	for _, g := range groups {
		for _, name := range g {
			m[name] = g[0] // canonical = first entry
		}
	}
	return m
}()

// matchesCodec checks if a codec matches any of the codec names,
// including common aliases.
func matchesCodec(codec string, codecs []string) bool {
	return matchesWithAliases(codec, codecs, codecAliasGroups)
}

func checkIntLimitation(value int, comparison string, values []string) bool {
	if len(values) == 0 {
		return true
	}

	switch comparison {
	case ComparisonLessThanEqual:
		limit, ok := parseInt(values[0])
		if !ok {
			return true
		}
		return value <= limit
	case ComparisonGreaterThanEqual:
		limit, ok := parseInt(values[0])
		if !ok {
			return true
		}
		return value >= limit
	case ComparisonEquals:
		for _, v := range values {
			if limit, ok := parseInt(v); ok && value == limit {
				return true
			}
		}
		return false
	case ComparisonNotEquals:
		for _, v := range values {
			if limit, ok := parseInt(v); ok && value == limit {
				return false
			}
		}
		return true
	}
	return true
}

// checkStringLimitation checks a string value against a limitation.
// Only Equals and NotEquals comparisons are meaningful for strings.
// LessThanEqual/GreaterThanEqual are not applicable and always pass.
func checkStringLimitation(value string, comparison string, values []string) bool {
	switch comparison {
	case ComparisonEquals:
		for _, v := range values {
			if strings.EqualFold(value, v) {
				return true
			}
		}
		return false
	case ComparisonNotEquals:
		for _, v := range values {
			if strings.EqualFold(value, v) {
				return false
			}
		}
		return true
	}
	return true
}

func parseInt(s string) (int, bool) {
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return 0, false
	}
	return v, true
}

func isLosslessFormat(format string) bool {
	switch strings.ToLower(format) {
	case "flac", "alac", "wav", "aiff", "ape", "wv", "tta", "tak", "shn", "dsd":
		return true
	}
	return false
}
