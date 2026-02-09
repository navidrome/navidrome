package subsonic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/transcode"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

// API-layer request structs for JSON unmarshaling (decoupled from core structs)

// clientInfoRequest represents client playback capabilities from the request body
type clientInfoRequest struct {
	Name                       string                      `json:"name,omitempty"`
	Platform                   string                      `json:"platform,omitempty"`
	MaxAudioBitrate            int                         `json:"maxAudioBitrate,omitempty"`
	MaxTranscodingAudioBitrate int                         `json:"maxTranscodingAudioBitrate,omitempty"`
	DirectPlayProfiles         []directPlayProfileRequest  `json:"directPlayProfiles,omitempty"`
	TranscodingProfiles        []transcodingProfileRequest `json:"transcodingProfiles,omitempty"`
	CodecProfiles              []codecProfileRequest       `json:"codecProfiles,omitempty"`
}

// directPlayProfileRequest describes a format the client can play directly
type directPlayProfileRequest struct {
	Containers       []string `json:"containers,omitempty"`
	AudioCodecs      []string `json:"audioCodecs,omitempty"`
	Protocols        []string `json:"protocols,omitempty"`
	MaxAudioChannels int      `json:"maxAudioChannels,omitempty"`
}

// transcodingProfileRequest describes a transcoding target the client supports
type transcodingProfileRequest struct {
	Container        string `json:"container,omitempty"`
	AudioCodec       string `json:"audioCodec,omitempty"`
	Protocol         string `json:"protocol,omitempty"`
	MaxAudioChannels int    `json:"maxAudioChannels,omitempty"`
}

// codecProfileRequest describes codec-specific limitations
type codecProfileRequest struct {
	Type        string              `json:"type,omitempty"`
	Name        string              `json:"name,omitempty"`
	Limitations []limitationRequest `json:"limitations,omitempty"`
}

// limitationRequest describes a specific codec limitation
type limitationRequest struct {
	Name       string   `json:"name,omitempty"`
	Comparison string   `json:"comparison,omitempty"`
	Values     []string `json:"values,omitempty"`
	Required   bool     `json:"required,omitempty"`
}

// toCoreClientInfo converts the API request struct to the transcode.ClientInfo struct.
// The OpenSubsonic spec uses bps for bitrate values; core uses kbps.
func (r *clientInfoRequest) toCoreClientInfo() *transcode.ClientInfo {
	ci := &transcode.ClientInfo{
		Name:                       r.Name,
		Platform:                   r.Platform,
		MaxAudioBitrate:            bpsToKbps(r.MaxAudioBitrate),
		MaxTranscodingAudioBitrate: bpsToKbps(r.MaxTranscodingAudioBitrate),
	}

	for _, dp := range r.DirectPlayProfiles {
		ci.DirectPlayProfiles = append(ci.DirectPlayProfiles, transcode.DirectPlayProfile{
			Containers:       dp.Containers,
			AudioCodecs:      dp.AudioCodecs,
			Protocols:        dp.Protocols,
			MaxAudioChannels: dp.MaxAudioChannels,
		})
	}

	for _, tp := range r.TranscodingProfiles {
		ci.TranscodingProfiles = append(ci.TranscodingProfiles, transcode.Profile{
			Container:        tp.Container,
			AudioCodec:       tp.AudioCodec,
			Protocol:         tp.Protocol,
			MaxAudioChannels: tp.MaxAudioChannels,
		})
	}

	for _, cp := range r.CodecProfiles {
		coreCP := transcode.CodecProfile{
			Type: cp.Type,
			Name: cp.Name,
		}
		for _, lim := range cp.Limitations {
			coreLim := transcode.Limitation{
				Name:       lim.Name,
				Comparison: lim.Comparison,
				Values:     lim.Values,
				Required:   lim.Required,
			}
			// Convert audioBitrate limitation values from bps to kbps
			if lim.Name == transcode.LimitationAudioBitrate {
				coreLim.Values = convertBitrateValues(lim.Values)
			}
			coreCP.Limitations = append(coreCP.Limitations, coreLim)
		}
		ci.CodecProfiles = append(ci.CodecProfiles, coreCP)
	}

	return ci
}

// bpsToKbps converts bits per second to kilobits per second (rounded).
func bpsToKbps(bps int) int {
	return (bps + 500) / 1000
}

// kbpsToBps converts kilobits per second to bits per second.
func kbpsToBps(kbps int) int {
	return kbps * 1000
}

// convertBitrateValues converts a slice of bps string values to kbps string values.
func convertBitrateValues(bpsValues []string) []string {
	result := make([]string, len(bpsValues))
	for i, v := range bpsValues {
		n, err := strconv.Atoi(v)
		if err == nil {
			result[i] = strconv.Itoa(bpsToKbps(n))
		} else {
			result[i] = v // preserve unparseable values as-is
		}
	}
	return result
}

// validate checks that all enum fields in the request contain valid values per the OpenSubsonic spec.
func (r *clientInfoRequest) validate() error {
	for _, dp := range r.DirectPlayProfiles {
		for _, p := range dp.Protocols {
			if !isValidProtocol(p) {
				return fmt.Errorf("invalid protocol: %s", p)
			}
		}
	}
	for _, tp := range r.TranscodingProfiles {
		if tp.Protocol != "" && !isValidProtocol(tp.Protocol) {
			return fmt.Errorf("invalid protocol: %s", tp.Protocol)
		}
	}
	for _, cp := range r.CodecProfiles {
		if !isValidCodecProfileType(cp.Type) {
			return fmt.Errorf("invalid codec profile type: %s", cp.Type)
		}
		for _, lim := range cp.Limitations {
			if !isValidLimitationName(lim.Name) {
				return fmt.Errorf("invalid limitation name: %s", lim.Name)
			}
			if !isValidComparison(lim.Comparison) {
				return fmt.Errorf("invalid comparison: %s", lim.Comparison)
			}
		}
	}
	return nil
}

var validProtocols = []string{
	transcode.ProtocolHTTP,
	transcode.ProtocolHLS,
}

func isValidProtocol(p string) bool {
	return slices.Contains(validProtocols, p)
}

var validCodecProfileTypes = []string{
	transcode.CodecProfileTypeAudio,
}

func isValidCodecProfileType(t string) bool {
	return slices.Contains(validCodecProfileTypes, t)
}

var validLimitationNames = []string{
	transcode.LimitationAudioChannels,
	transcode.LimitationAudioBitrate,
	transcode.LimitationAudioProfile,
	transcode.LimitationAudioSamplerate,
	transcode.LimitationAudioBitdepth,
}

func isValidLimitationName(n string) bool {
	return slices.Contains(validLimitationNames, n)
}

var validComparisons = []string{
	transcode.ComparisonEquals,
	transcode.ComparisonNotEquals,
	transcode.ComparisonLessThanEqual,
	transcode.ComparisonGreaterThanEqual,
}

func isValidComparison(c string) bool {
	return slices.Contains(validComparisons, c)
}

// GetTranscodeDecision handles the OpenSubsonic getTranscodeDecision endpoint.
// It receives client capabilities and returns a decision on whether to direct play or transcode.
func (api *Router) GetTranscodeDecision(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return nil, nil
	}

	ctx := r.Context()
	p := req.Params(r)

	mediaID, err := p.String("mediaId")
	if err != nil {
		return nil, newError(responses.ErrorMissingParameter, "missing required parameter: mediaId")
	}

	mediaType, err := p.String("mediaType")
	if err != nil {
		return nil, newError(responses.ErrorMissingParameter, "missing required parameter: mediaType")
	}

	// Only support songs for now
	if mediaType != "song" {
		return nil, newError(responses.ErrorGeneric, "mediaType '%s' is not yet supported", mediaType)
	}

	// Parse and validate ClientInfo from request body (required per OpenSubsonic spec)
	var clientInfoReq clientInfoRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	if err := json.NewDecoder(r.Body).Decode(&clientInfoReq); err != nil {
		return nil, newError(responses.ErrorGeneric, "invalid JSON request body")
	}
	if err := clientInfoReq.validate(); err != nil {
		return nil, newError(responses.ErrorGeneric, "%v", err)
	}
	clientInfo := clientInfoReq.toCoreClientInfo()

	// Get media file
	mf, err := api.ds.MediaFile(ctx).Get(mediaID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, newError(responses.ErrorDataNotFound, "media file not found: %s", mediaID)
		}
		return nil, newError(responses.ErrorGeneric, "error retrieving media file: %v", err)
	}

	// Make the decision
	decision, err := api.transcodeDecision.MakeDecision(ctx, mf, clientInfo)
	if err != nil {
		return nil, newError(responses.ErrorGeneric, "failed to make transcode decision: %v", err)
	}

	// Only create a token when there is a valid playback path
	var transcodeParams string
	if decision.CanDirectPlay || decision.CanTranscode {
		transcodeParams, err = api.transcodeDecision.CreateTranscodeParams(decision)
		if err != nil {
			return nil, newError(responses.ErrorGeneric, "failed to create transcode token: %v", err)
		}
	}

	// Build response (convert kbps from core to bps for the API)
	response := newResponse()
	response.TranscodeDecision = &responses.TranscodeDecision{
		CanDirectPlay:    decision.CanDirectPlay,
		CanTranscode:     decision.CanTranscode,
		TranscodeReasons: decision.TranscodeReasons,
		ErrorReason:      decision.ErrorReason,
		TranscodeParams:  transcodeParams,
		SourceStream: &responses.StreamDetails{
			Protocol:        "http",
			Container:       decision.SourceStream.Container,
			Codec:           decision.SourceStream.Codec,
			AudioBitrate:    int32(kbpsToBps(decision.SourceStream.Bitrate)),
			AudioProfile:    decision.SourceStream.Profile,
			AudioSamplerate: int32(decision.SourceStream.SampleRate),
			AudioBitdepth:   int32(decision.SourceStream.BitDepth),
			AudioChannels:   int32(decision.SourceStream.Channels),
		},
	}

	if decision.TranscodeStream != nil {
		response.TranscodeDecision.TranscodeStream = &responses.StreamDetails{
			Protocol:        "http",
			Container:       decision.TranscodeStream.Container,
			Codec:           decision.TranscodeStream.Codec,
			AudioBitrate:    int32(kbpsToBps(decision.TranscodeStream.Bitrate)),
			AudioProfile:    decision.TranscodeStream.Profile,
			AudioSamplerate: int32(decision.TranscodeStream.SampleRate),
			AudioBitdepth:   int32(decision.TranscodeStream.BitDepth),
			AudioChannels:   int32(decision.TranscodeStream.Channels),
		}
	}

	return response, nil
}

// GetTranscodeStream handles the OpenSubsonic getTranscodeStream endpoint.
// It streams media using the decision encoded in the transcodeParams JWT token.
func (api *Router) GetTranscodeStream(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	p := req.Params(r)

	mediaID, err := p.String("mediaId")
	if err != nil {
		return nil, newError(responses.ErrorMissingParameter, "missing required parameter: mediaId")
	}

	mediaType, err := p.String("mediaType")
	if err != nil {
		return nil, newError(responses.ErrorMissingParameter, "missing required parameter: mediaType")
	}

	transcodeParams, err := p.String("transcodeParams")
	if err != nil {
		return nil, newError(responses.ErrorMissingParameter, "missing required parameter: transcodeParams")
	}

	// Only support songs for now
	if mediaType != "song" {
		return nil, newError(responses.ErrorGeneric, "mediaType '%s' is not yet supported", mediaType)
	}

	// Parse and validate the token
	params, err := api.transcodeDecision.ParseTranscodeParams(transcodeParams)
	if err != nil {
		log.Warn(ctx, "Failed to parse transcode token", err)
		return nil, newError(responses.ErrorDataNotFound, "invalid or expired transcodeParams token")
	}

	// Verify mediaId matches token
	if params.MediaID != mediaID {
		return nil, newError(responses.ErrorDataNotFound, "mediaId does not match token")
	}

	// Build streaming parameters from the token
	streamReq := core.StreamRequest{ID: mediaID, Offset: p.IntOr("offset", 0)}
	if !params.DirectPlay && params.TargetFormat != "" {
		streamReq.Format = params.TargetFormat
		streamReq.BitRate = params.TargetBitrate // Already in kbps, matching the streamer
		streamReq.SampleRate = params.TargetSampleRate
		streamReq.BitDepth = params.TargetBitDepth
		streamReq.Channels = params.TargetChannels
	}

	// Create stream
	stream, err := api.streamer.NewStream(ctx, streamReq)
	if err != nil {
		return nil, err
	}

	// Make sure the stream will be closed at the end
	defer func() {
		if err := stream.Close(); err != nil && log.IsGreaterOrEqualTo(log.LevelDebug) {
			log.Error("Error closing stream", "id", mediaID, "file", stream.Name(), err)
		}
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")

	api.serveStream(ctx, w, r, stream, mediaID)

	return nil, nil
}
