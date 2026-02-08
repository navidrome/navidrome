package transcode

import (
	"context"
	"slices"
	"strconv"
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

// Decider is the core service interface for making transcoding decisions
type Decider interface {
	MakeDecision(ctx context.Context, mf *model.MediaFile, clientInfo *ClientInfo) (*Decision, error)
	CreateTranscodeParams(decision *Decision) (string, error)
	ParseTranscodeParams(token string) (*Params, error)
}

// ClientInfo represents client playback capabilities.
// All bitrate values are in kilobits per second (kbps)
type ClientInfo struct {
	Name                       string
	Platform                   string
	MaxAudioBitrate            int
	MaxTranscodingAudioBitrate int
	DirectPlayProfiles         []DirectPlayProfile
	TranscodingProfiles        []Profile
	CodecProfiles              []CodecProfile
}

// DirectPlayProfile describes a format the client can play directly
type DirectPlayProfile struct {
	Containers       []string
	AudioCodecs      []string
	Protocols        []string
	MaxAudioChannels int
}

// Profile describes a transcoding target the client supports
type Profile struct {
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
	TargetSampleRate int
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

// Params contains the parameters extracted from a transcode token.
// TargetBitrate is in kilobits per second (kbps).
type Params struct {
	MediaID          string
	DirectPlay       bool
	TargetFormat     string
	TargetBitrate    int
	TargetChannels   int
	TargetSampleRate int
}

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
		SampleRate: dsdToPCMSampleRate(mf.SampleRate, mf.AudioCodec()),
		Channels:   mf.Channels,
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
	tc, err := s.ds.Transcoding(ctx).FindByFormat(targetFormat)
	if (err != nil || tc == nil) && profile.AudioCodec != "" && !strings.EqualFold(targetFormat, profile.AudioCodec) {
		codec := strings.ToLower(profile.AudioCodec)
		log.Trace(ctx, "No transcoding config for container, trying audioCodec", "container", targetFormat, "audioCodec", codec)
		tc, err = s.ds.Transcoding(ctx).FindByFormat(codec)
		if err == nil && tc != nil {
			targetFormat = codec
		}
	}
	if err != nil || tc == nil {
		log.Trace(ctx, "Skipping transcoding profile: no transcoding config", "targetFormat", targetFormat)
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
	}
	return auth.CreateExpiringPublicToken(exp, claims)
}

func (s *deciderService) ParseTranscodeParams(token string) (*Params, error) {
	claims, err := auth.Validate(token)
	if err != nil {
		return nil, err
	}

	params := &Params{}
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
	if sr, ok := claims["sr"].(float64); ok {
		params.TargetSampleRate = int(sr)
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

// dsdToPCMSampleRate converts a DSD sample rate to its PCM-equivalent rate (÷8).
// DSD64=2822400→352800, DSD128=5644800→705600, etc.
// For non-DSD codecs, returns the rate unchanged.
func dsdToPCMSampleRate(sampleRate int, codec string) int {
	if strings.EqualFold(codec, "dsd") && sampleRate > 0 {
		return sampleRate / 8
	}
	return sampleRate
}

// codecFixedOutputSampleRate returns the mandatory output sample rate for codecs
// that always resample regardless of input (e.g., Opus always outputs 48000Hz).
// Returns 0 if the codec has no fixed output rate.
func codecFixedOutputSampleRate(codec string) int {
	switch strings.ToLower(codec) {
	case "opus":
		return 48000
	}
	return 0
}

// codecMaxSampleRate returns the hard maximum output sample rate for a codec.
// Returns 0 if the codec has no hard limit.
func codecMaxSampleRate(codec string) int {
	switch strings.ToLower(codec) {
	case "mp3":
		return 48000
	case "aac":
		return 96000
	}
	return 0
}
