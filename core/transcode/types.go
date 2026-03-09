package transcode

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/navidrome/navidrome/model"
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

// Decider is the core service interface for making transcoding decisions
type Decider interface {
	MakeDecision(ctx context.Context, mf *model.MediaFile, clientInfo *ClientInfo, opts DecisionOptions) (*Decision, error)
	ResolveStream(ctx context.Context, mf *model.MediaFile, reqFormat string, reqBitRate int, offset int) StreamRequest
	CreateTranscodeParams(decision *Decision) (string, error)
	ValidateTranscodeParams(ctx context.Context, token string, mediaID string) (*Params, *model.MediaFile, error)
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

// toClaimsMap converts a Decision into a JWT claims map for token encoding.
// Only non-zero transcode fields are included.
func (d *Decision) toClaimsMap() map[string]any {
	m := map[string]any{
		"mid":             d.MediaID,
		"ua":              d.SourceUpdatedAt.Truncate(time.Second).Unix(),
		jwt.ExpirationKey: time.Now().Add(tokenTTL).UTC().Unix(),
	}
	if d.CanDirectPlay {
		m["dp"] = true
	}
	if d.CanTranscode && d.TargetFormat != "" {
		m["f"] = d.TargetFormat
		if d.TargetBitrate != 0 {
			m["b"] = d.TargetBitrate
		}
		if d.TargetChannels != 0 {
			m["ch"] = d.TargetChannels
		}
		if d.TargetSampleRate != 0 {
			m["sr"] = d.TargetSampleRate
		}
		if d.TargetBitDepth != 0 {
			m["bd"] = d.TargetBitDepth
		}
	}
	return m
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

// Params contains the parameters extracted from a transcode token.
// TargetBitrate is in kilobits per second (kbps).
type Params struct {
	MediaID          string
	DirectPlay       bool
	TargetFormat     string
	TargetBitrate    int
	TargetChannels   int
	TargetSampleRate int
	TargetBitDepth   int
	SourceUpdatedAt  time.Time
}

// paramsFromToken extracts and validates Params from a parsed JWT token.
// Returns an error if required claims (media ID, source timestamp) are missing.
func paramsFromToken(token jwt.Token) (*Params, error) {
	var p Params
	var mid string
	if err := token.Get("mid", &mid); err == nil {
		p.MediaID = mid
	}
	if p.MediaID == "" {
		return nil, fmt.Errorf("%w: missing media ID", ErrTokenInvalid)
	}

	var dp bool
	if err := token.Get("dp", &dp); err == nil {
		p.DirectPlay = dp
	}

	ua := getIntClaim(token, "ua")
	if ua != 0 {
		p.SourceUpdatedAt = time.Unix(int64(ua), 0)
	}
	if p.SourceUpdatedAt.IsZero() {
		return nil, fmt.Errorf("%w: missing source timestamp", ErrTokenInvalid)
	}

	var f string
	if err := token.Get("f", &f); err == nil {
		p.TargetFormat = f
	}
	p.TargetBitrate = getIntClaim(token, "b")
	p.TargetChannels = getIntClaim(token, "ch")
	p.TargetSampleRate = getIntClaim(token, "sr")
	p.TargetBitDepth = getIntClaim(token, "bd")
	return &p, nil
}

// getIntClaim extracts an int claim from a JWT token, handling the case where
// the value may be stored as int64 or float64 (common in JSON-based JWT libraries).
func getIntClaim(token jwt.Token, key string) int {
	var v int
	if err := token.Get(key, &v); err == nil {
		return v
	}
	var v64 int64
	if err := token.Get(key, &v64); err == nil {
		return int(v64)
	}
	var f float64
	if err := token.Get(key, &f); err == nil {
		return int(f)
	}
	return 0
}
