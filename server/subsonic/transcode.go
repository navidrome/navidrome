package subsonic

import (
	"encoding/json"
	"net/http"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

// API-layer request structs for JSON unmarshaling (decoupled from core structs)

// clientInfoRequest represents client playback capabilities from the request body
type clientInfoRequest struct {
	Name                       string                  `json:"name,omitempty"`
	Platform                   string                  `json:"platform,omitempty"`
	MaxAudioBitrate            int                     `json:"maxAudioBitrate,omitempty"`
	MaxTranscodingAudioBitrate int                     `json:"maxTranscodingAudioBitrate,omitempty"`
	DirectPlayProfiles         []directPlayProfileReq  `json:"directPlayProfiles,omitempty"`
	TranscodingProfiles        []transcodingProfileReq `json:"transcodingProfiles,omitempty"`
	CodecProfiles              []codecProfileReq       `json:"codecProfiles,omitempty"`
}

// directPlayProfileReq describes a format the client can play directly
type directPlayProfileReq struct {
	Containers       []string `json:"containers,omitempty"`
	AudioCodecs      []string `json:"audioCodecs,omitempty"`
	Protocols        []string `json:"protocols,omitempty"`
	MaxAudioChannels int      `json:"maxAudioChannels,omitempty"`
}

// transcodingProfileReq describes a transcoding target the client supports
type transcodingProfileReq struct {
	Container        string `json:"container,omitempty"`
	AudioCodec       string `json:"audioCodec,omitempty"`
	Protocol         string `json:"protocol,omitempty"`
	MaxAudioChannels int    `json:"maxAudioChannels,omitempty"`
}

// codecProfileReq describes codec-specific limitations
type codecProfileReq struct {
	Type        string          `json:"type,omitempty"`
	Name        string          `json:"name,omitempty"`
	Limitations []limitationReq `json:"limitations,omitempty"`
}

// limitationReq describes a specific codec limitation
type limitationReq struct {
	Property  string `json:"property,omitempty"`
	Condition string `json:"condition,omitempty"`
	Value     string `json:"value,omitempty"`
}

// toCore converts the API request struct to the core ClientInfo struct
func (r *clientInfoRequest) toCore() *core.ClientInfo {
	ci := &core.ClientInfo{
		Name:                       r.Name,
		Platform:                   r.Platform,
		MaxAudioBitrate:            r.MaxAudioBitrate,
		MaxTranscodingAudioBitrate: r.MaxTranscodingAudioBitrate,
	}

	for _, dp := range r.DirectPlayProfiles {
		ci.DirectPlayProfiles = append(ci.DirectPlayProfiles, core.DirectPlayProfile{
			Containers:       dp.Containers,
			AudioCodecs:      dp.AudioCodecs,
			Protocols:        dp.Protocols,
			MaxAudioChannels: dp.MaxAudioChannels,
		})
	}

	for _, tp := range r.TranscodingProfiles {
		ci.TranscodingProfiles = append(ci.TranscodingProfiles, core.TranscodingProfile{
			Container:        tp.Container,
			AudioCodec:       tp.AudioCodec,
			Protocol:         tp.Protocol,
			MaxAudioChannels: tp.MaxAudioChannels,
		})
	}

	for _, cp := range r.CodecProfiles {
		coreCP := core.CodecProfile{
			Type: cp.Type,
			Name: cp.Name,
		}
		for _, lim := range cp.Limitations {
			coreCP.Limitations = append(coreCP.Limitations, core.Limitation{
				Property:  lim.Property,
				Condition: lim.Condition,
				Value:     lim.Value,
			})
		}
		ci.CodecProfiles = append(ci.CodecProfiles, coreCP)
	}

	return ci
}

// GetTranscodeDecision handles the OpenSubsonic getTranscodeDecision endpoint.
// It receives client capabilities and returns a decision on whether to direct play or transcode.
func (api *Router) GetTranscodeDecision(_ http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
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

	// Parse ClientInfo from request body
	var clientInfoReq clientInfoRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&clientInfoReq); err != nil {
			log.Debug(ctx, "Failed to parse client info from body", err)
			// Continue with empty client info - will likely result in no compatible profile
		}
	}
	clientInfo := clientInfoReq.toCore()

	// Get media file
	mf, err := api.ds.MediaFile(ctx).Get(mediaID)
	if err != nil {
		return nil, newError(responses.ErrorDataNotFound, "media file not found: %s", mediaID)
	}

	// Make the decision
	decision, err := api.transcodeDecision.MakeDecision(ctx, mf, clientInfo)
	if err != nil {
		return nil, newError(responses.ErrorGeneric, "failed to make transcode decision: %v", err)
	}

	// Create token
	transcodeParams, err := api.transcodeDecision.CreateToken(decision)
	if err != nil {
		return nil, newError(responses.ErrorGeneric, "failed to create transcode token: %v", err)
	}

	// Build response
	response := newResponse()
	response.TranscodeDecision = &responses.TranscodeDecision{
		CanDirectPlay:    decision.CanDirectPlay,
		CanTranscode:     decision.CanTranscode,
		TranscodeReasons: decision.TranscodeReasons,
		ErrorReason:      decision.ErrorReason,
		TranscodeParams:  transcodeParams,
		SourceStream: &responses.StreamDetails{
			Container:  decision.SourceStream.Container,
			Codec:      decision.SourceStream.Codec,
			Bitrate:    int32(decision.SourceStream.Bitrate),
			SampleRate: int32(decision.SourceStream.SampleRate),
			BitDepth:   int32(decision.SourceStream.BitDepth),
			Channels:   int32(decision.SourceStream.Channels),
			Duration:   int32(decision.SourceStream.Duration),
			Size:       decision.SourceStream.Size,
			IsLossless: decision.SourceStream.IsLossless,
		},
	}

	if decision.TranscodeStream != nil {
		response.TranscodeDecision.TranscodeStream = &responses.StreamDetails{
			Container:  decision.TranscodeStream.Container,
			Codec:      decision.TranscodeStream.Codec,
			Bitrate:    int32(decision.TranscodeStream.Bitrate),
			SampleRate: int32(decision.TranscodeStream.SampleRate),
			BitDepth:   int32(decision.TranscodeStream.BitDepth),
			Channels:   int32(decision.TranscodeStream.Channels),
			IsLossless: decision.TranscodeStream.IsLossless,
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
	params, err := api.transcodeDecision.ParseToken(transcodeParams)
	if err != nil {
		log.Debug(ctx, "Failed to parse transcode token", err)
		return nil, newError(responses.ErrorDataNotFound, "invalid or expired transcodeParams token")
	}

	// Verify mediaId matches token
	if params.MediaID != mediaID {
		return nil, newError(responses.ErrorDataNotFound, "mediaId does not match token")
	}

	// Determine streaming parameters
	format := ""
	maxBitRate := 0
	if !params.DirectPlay && params.TargetFormat != "" {
		format = params.TargetFormat
		maxBitRate = params.TargetBitrate
	}

	// Get offset parameter
	offset := p.IntOr("offset", 0)

	// Create stream
	stream, err := api.streamer.NewStream(ctx, mediaID, format, maxBitRate, offset)
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
