package jellyfin

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
)

// mediaFileForRequest resolves {itemId} to a MediaFile and verifies the user has access to its
// library, writing 404 (never 403, to avoid an existence oracle) and returning ok=false otherwise.
// Shared by getPlaybackInfo and streamAudio so a guessed id can't probe or stream another library.
func (api *Router) mediaFileForRequest(w http.ResponseWriter, r *http.Request) (*model.MediaFile, bool) {
	ctx := r.Context()
	id := api.resolveItemID(ctx, dto.DecodeID(chi.URLParam(r, "itemId")))
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

// getPlaybackInfo answers /Items/{itemId}/PlaybackInfo with a single MediaSource for direct
// playback. Format negotiation happens later in streamAudio (like Subsonic defers it to /stream).
func (api *Router) getPlaybackInfo(w http.ResponseWriter, r *http.Request) {
	mf, ok := api.mediaFileForRequest(w, r)
	if !ok {
		return
	}
	src := dto.MediaSourceFromMediaFile(*mf)
	// Embed the caller's token in the stream URL: Jellify's native player fetches TranscodingUrl
	// verbatim without an auth header, so a non-self-authenticating URL would 401. Direct-play clients
	// (Finamp) build their own /File?ApiKey URL and ignore this. Include the /jellyfin mount prefix so
	// a client resolving it as an absolute host path still hits the mounted router.
	if token := tokenFromRequest(r); token != "" {
		src.TranscodingSubProtocol = "http"
		src.TranscodingUrl = consts.URLPathJellyfinAPI + "/Audio/" + src.Id + "/universal?static=true&api_key=" + url.QueryEscape(token)
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
		// The /stream.{container} route form carries the format as a path segment, not a query param.
		format = chi.URLParam(r, "container")
	}
	if format == "" {
		// Jellyfin's audioCodec param names the target codec when no container is given.
		format = p.StringOr("audiocodec", "")
	}
	if p.BoolOr("static", false) {
		format = "raw"
	}

	// Bitrate params are bits/sec by Jellyfin convention; ResolveRequest expects kbps.
	bitRate := p.IntOr("audiobitrate", 0) / 1000
	if bitRate == 0 {
		bitRate = p.IntOr("maxstreamingbitrate", 0) / 1000
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

// streamHls serves /Audio/{itemId}/main.m3u8 (Finamp's transcoding mode) as a single-segment VOD
// playlist whose one segment is the progressive transcode endpoint, reusing that whole pipeline.
// Trade-off: seeking re-reads from the start, like Subsonic transcoded streams.
func (api *Router) streamHls(w http.ResponseWriter, r *http.Request) {
	mf, ok := api.mediaFileForRequest(w, r)
	if !ok {
		return
	}
	p := req.Params(r)

	// HLS packed audio can only carry ADTS/AAC or MP3; other codecs fall back to aac. A forced
	// transcoding wins verbatim — its override rewrites the segment anyway, and the playlist must match.
	codec := strings.ToLower(p.StringOr("audiocodec", ""))
	if codec != "mp3" {
		codec = "aac"
	}
	if trc, ok := request.TranscodingFrom(r.Context()); ok && trc.TargetFormat != "" {
		codec = strings.ToLower(trc.TargetFormat)
	}

	// Relative to the playlist URL. HLS fetches drop auth headers, so the token rides in the query.
	segment := "stream." + codec
	q := url.Values{}
	if token := tokenFromRequest(r); token != "" {
		q.Set("api_key", token)
	}
	if bitRate := p.IntOr("audiobitrate", 0); bitRate > 0 {
		q.Set("audioBitRate", strconv.Itoa(bitRate))
	}
	if len(q) > 0 {
		segment += "?" + q.Encode()
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	//nolint:gosec // not HTML; the only tainted value is query-escaped
	fmt.Fprintf(w, "#EXTM3U\n"+
		"#EXT-X-VERSION:3\n"+
		"#EXT-X-PLAYLIST-TYPE:VOD\n"+
		"#EXT-X-TARGETDURATION:%d\n"+
		"#EXT-X-MEDIA-SEQUENCE:0\n"+
		"#EXTINF:%.3f,\n"+
		"%s\n"+
		"#EXT-X-ENDLIST\n",
		int(math.Ceil(float64(mf.Duration))), mf.Duration, segment)
}

// streamFile serves /Items/{itemId}/File and /Download, Jellyfin's direct-file endpoints. Some
// clients (Finamp's just_audio engine) fetch playback audio here instead of /Audio/{id}/stream, so
// it must always resolve to direct play ("raw"), never a forced transcode.
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
