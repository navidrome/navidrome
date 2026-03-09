package transcode

import (
	"errors"
	"time"
)

var (
	ErrTokenInvalid  = errors.New("invalid or expired transcode token")
	ErrMediaNotFound = errors.New("media file not found")
	ErrTokenStale    = errors.New("transcode token is stale: media file has changed")
)

// DecisionOptions controls optional behavior of MakeDecision.
type DecisionOptions struct {
	// SkipProbe prevents MakeDecision from running ffprobe on the media file.
	// When true, source stream details are derived from tag metadata only.
	SkipProbe bool
}

// StreamRequest contains the resolved parameters for creating a media stream.
type StreamRequest struct {
	ID         string
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
	TargetBitDepth   int
	SourceStream     StreamDetails
	SourceUpdatedAt  time.Time
	TranscodeStream  *StreamDetails
}

// StreamDetails describes audio stream properties.
// Bitrate is in kilobits per second (kbps).
type StreamDetails struct {
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
