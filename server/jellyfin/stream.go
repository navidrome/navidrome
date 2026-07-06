package jellyfin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
)

// mediaFileForRequest resolves the {itemId} path param to a MediaFile and verifies the
// current user has access to its library, writing a 404 response (never 403, to avoid an
// existence oracle) and returning ok=false if either check fails. Shared by getPlaybackInfo
// and streamAudio so a guessed id can't be used to probe -- or stream -- another library's
// content, mirroring the same gate applied to getItem in items.go.
func (api *Router) mediaFileForRequest(w http.ResponseWriter, r *http.Request) (*model.MediaFile, bool) {
	ctx := r.Context()
	id := chi.URLParam(r, "itemId")
	mf, err := api.ds.MediaFile(ctx).Get(id)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return nil, false
	}
	u, _ := request.UserFrom(ctx)
	if !u.HasLibraryAccess(mf.LibraryID) {
		http.Error(w, "Not Found", http.StatusNotFound)
		return nil, false
	}
	return mf, true
}

// getPlaybackInfo answers POST/GET /Items/{itemId}/PlaybackInfo with a single MediaSource
// describing direct playback of the source file. Actual format negotiation happens at stream
// time in streamAudio, mirroring how the Subsonic API defers that decision to /stream.
func (api *Router) getPlaybackInfo(w http.ResponseWriter, r *http.Request) {
	mf, ok := api.mediaFileForRequest(w, r)
	if !ok {
		return
	}
	src := dto.MediaSourceInfo{
		Id:                   mf.ID,
		Protocol:             "Http",
		Container:            mf.Suffix,
		Size:                 mf.Size,
		Name:                 mf.Title,
		Type:                 "Default",
		RunTimeTicks:         dto.TicksFromSeconds(mf.Duration),
		SupportsDirectPlay:   true,
		SupportsDirectStream: true,
		SupportsTranscoding:  true,
	}
	api.ok(w, r, dto.PlaybackInfoResponse{MediaSources: []dto.MediaSourceInfo{src}, PlaySessionId: mf.ID})
}

// streamAudio serves /Audio/{itemId}/stream[.container] and /Audio/{itemId}/universal,
// reusing the same transcode-decision + streaming pipeline as the Subsonic /stream endpoint.
func (api *Router) streamAudio(w http.ResponseWriter, r *http.Request) {
	mf, ok := api.mediaFileForRequest(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	p := req.Params(r)

	format := p.StringOr("container", "")
	if format == "" {
		// The /stream.{container} route form carries the format as a path segment,
		// not a query param.
		format = chi.URLParam(r, "container")
	}
	if p.BoolOr("static", false) {
		format = "raw"
	}

	bitRate := p.IntOr("audioBitRate", 0)
	if bitRate == 0 {
		// maxStreamingBitrate is bits/sec by Jellyfin convention; ResolveRequest expects kbps.
		bitRate = p.IntOr("maxStreamingBitrate", 0) / 1000
	}

	streamReq := api.transcodeDecider.ResolveRequest(ctx, mf, format, bitRate, 0)
	s, err := api.streamer.NewStream(ctx, mf, streamReq)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	defer s.Close()
	if _, err := s.Serve(ctx, w, r); err != nil {
		log.Error(ctx, "Jellyfin API: error streaming", "id", mf.ID, err)
	}
}

// streamFile serves /Items/{itemId}/File and /Items/{itemId}/Download, Jellyfin's direct-file
// endpoints. Real Jellyfin returns the original media file unmodified here; some clients (e.g.
// Finamp's just_audio engine) fetch playback audio from this URL instead of /Audio/{id}/stream,
// so it must always resolve to direct play ("raw"), never a forced transcode.
func (api *Router) streamFile(w http.ResponseWriter, r *http.Request) {
	mf, ok := api.mediaFileForRequest(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	streamReq := api.transcodeDecider.ResolveRequest(ctx, mf, "raw", 0, 0)
	s, err := api.streamer.NewStream(ctx, mf, streamReq)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	defer s.Close()
	if _, err := s.Serve(ctx, w, r); err != nil {
		log.Error(ctx, "Jellyfin API: error streaming", "id", mf.ID, err)
	}
}
