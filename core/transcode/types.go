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

// Decider is the core service interface for making transcoding decisions
type Decider interface {
	MakeDecision(ctx context.Context, mf *model.MediaFile, clientInfo *ClientInfo) (*Decision, error)
	CreateTranscodeParams(decision *Decision) (string, error)
	ParseTranscodeParams(token string) (*Params, error)
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

	var ua int64
	if err := token.Get("ua", &ua); err == nil {
		p.SourceUpdatedAt = time.Unix(ua, 0)
	} else {
		var uaf float64
		if err := token.Get("ua", &uaf); err == nil {
			p.SourceUpdatedAt = time.Unix(int64(uaf), 0)
		}
	}
	if p.SourceUpdatedAt.IsZero() {
		return nil, fmt.Errorf("%w: missing source timestamp", ErrTokenInvalid)
	}

	var f string
	if err := token.Get("f", &f); err == nil {
		p.TargetFormat = f
	}
	if err := token.Get("b", &p.TargetBitrate); err != nil {
		var bf float64
		if err := token.Get("b", &bf); err == nil {
			p.TargetBitrate = int(bf)
		}
	}
	if err := token.Get("ch", &p.TargetChannels); err != nil {
		var chf float64
		if err := token.Get("ch", &chf); err == nil {
			p.TargetChannels = int(chf)
		}
	}
	if err := token.Get("sr", &p.TargetSampleRate); err != nil {
		var srf float64
		if err := token.Get("sr", &srf); err == nil {
			p.TargetSampleRate = int(srf)
		}
	}
	if err := token.Get("bd", &p.TargetBitDepth); err != nil {
		var bdf float64
		if err := token.Get("bd", &bdf); err == nil {
			p.TargetBitDepth = int(bdf)
		}
	}
	return &p, nil
}
