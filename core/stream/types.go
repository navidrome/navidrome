package stream

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrTokenInvalid = errors.New("invalid or expired transcode token")
	ErrTokenStale   = errors.New("transcode token is stale: media file has changed")
)

// TranscodeOptions controls optional behavior of MakeTranscodeDecision.
type TranscodeOptions struct {
	// SkipProbe prevents MakeTranscodeDecision from running ffprobe on the media file.
	// When true, source stream details are derived from tag metadata only.
	SkipProbe bool
}

// Request contains the resolved parameters for creating a media stream.
type Request struct {
	Format     string
	BitRate    int // kbps
	SampleRate int
	BitDepth   int
	Channels   int
	Offset     int // seconds
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

// CapBitrate lowers the client's declared audio bitrate limits to maxKbps,
// never raising them. A zero limit means "unlimited" and is set to maxKbps.
// Returns true if anything changed. No-op when maxKbps <= 0.
func (ci *ClientInfo) CapBitrate(maxKbps int) bool {
	if maxKbps <= 0 {
		return false
	}
	changed := false
	if ci.MaxAudioBitrate == 0 || maxKbps < ci.MaxAudioBitrate {
		ci.MaxAudioBitrate = maxKbps
		changed = true
	}
	if ci.MaxTranscodingAudioBitrate == 0 || maxKbps < ci.MaxTranscodingAudioBitrate {
		ci.MaxTranscodingAudioBitrate = maxKbps
		changed = true
	}
	return changed
}

// ForceFormat narrows the client to transcoding to targetFormat and suppresses
// direct play, but only if the client already declares a profile for that
// format. All matching profiles are kept so negotiation can still pick among
// them (e.g. by protocol). Returns false (no-op) when targetFormat is empty or
// unsupported.
func (ci *ClientInfo) ForceFormat(targetFormat string) bool {
	if targetFormat == "" {
		return false
	}
	var matched []Profile
	for i := range ci.TranscodingProfiles {
		// matchesContainer is alias-aware, so a forced "oga" (legacy Opus
		// target_format) still matches a resolved "opus" profile.
		if _, format := resolveTargetFormat(&ci.TranscodingProfiles[i]); matchesContainer(format, []string{targetFormat}) {
			matched = append(matched, ci.TranscodingProfiles[i])
		}
	}
	if len(matched) == 0 {
		return false
	}
	ci.TranscodingProfiles = matched
	ci.DirectPlayProfiles = nil
	return true
}

// DirectPlayProfile describes a format the client can play directly
type DirectPlayProfile struct {
	Containers       []string
	AudioCodecs      []string
	Protocols        []string
	MaxAudioChannels int
}

func (p DirectPlayProfile) String() string {
	containers := strings.Join(p.Containers, ",")
	if containers == "" {
		containers = "*"
	}
	codecs := strings.Join(p.AudioCodecs, ",")
	if codecs == "" {
		return "[" + containers + "]"
	}
	return "[" + containers + "/" + codecs + "]"
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

// TranscodeDecision represents the internal decision result.
// All bitrate values are in kilobits per second (kbps).
type TranscodeDecision struct {
	MediaID          string
	CanDirectPlay    bool
	CanTranscode     bool
	TranscodeReasons []string
	ErrorReason      string
	TargetFormat     string
	TargetBitrate    int
	TargetChannels   int
	TargetSampleRate int
	TargetBitDepth   int
	SourceStream     Details
	SourceUpdatedAt  time.Time
	TranscodeStream  *Details
}

// Details describes audio stream properties.
// Bitrate is in kilobits per second (kbps).
type Details struct {
	Container  string
	Codec      string
	Profile    string // Audio profile (e.g., "LC", "HE-AACv2"). Populated from ffprobe data.
	Bitrate    int
	SampleRate int
	BitDepth   int
	Channels   int
	Duration   float32
	Size       int64
	IsLossless bool
}
