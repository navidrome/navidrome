package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

// playbackReport is the subset of Jellyfin's PlaybackStartInfo/PlaybackProgressInfo
// fields Navidrome needs to keep its playback/scrobbling state in sync.
type playbackReport struct {
	ItemId        string `json:"ItemId"`
	PositionTicks int64  `json:"PositionTicks"`
	IsPaused      bool   `json:"IsPaused"`
}

// decodeReport reads the playback report body. ItemId falls back to a query
// param, as some clients send it there instead of (or in addition to) the JSON body.
// ItemId is decoded here since it flows straight into scrobbler lookups by raw media file id.
func decodeReport(r *http.Request) playbackReport {
	var body playbackReport
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.ItemId == "" {
		body.ItemId = r.URL.Query().Get("ItemId")
	}
	body.ItemId = dto.DecodeID(body.ItemId)
	return body
}

// clientIdentity returns the scrobbler cache key/display name for the caller's
// player. Both are zero values if withPlayer could not resolve a player.
func clientIdentity(ctx context.Context) (id, name string) {
	player, _ := request.PlayerFrom(ctx)
	return player.ID, player.Client
}

// reportPlaybackStart handles POST /Sessions/Playing, sent once when a client
// starts playing an item.
//
// Access control: these Sessions endpoints report the caller's own playback
// history and are never used to look up or expose content, so unlike browse/stream
// endpoints they are intentionally not library-access-gated.
func (api *Router) reportPlaybackStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body := decodeReport(r)
	clientId, clientName := clientIdentity(ctx)
	err := api.scrobbler.ReportPlayback(ctx, scrobbler.ReportPlaybackParams{
		MediaId:      body.ItemId,
		PositionMs:   body.PositionTicks / 10_000,
		State:        scrobbler.StatePlaying,
		PlaybackRate: 1.0,
		ClientId:     clientId,
		ClientName:   clientName,
	})
	if err != nil {
		log.Warn(ctx, "Jellyfin API: report playback start failed", "id", body.ItemId, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// reportPlaybackProgress handles POST /Sessions/Playing/Progress, sent periodically
// (and on pause/resume/seek) while a client keeps playing an item.
func (api *Router) reportPlaybackProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body := decodeReport(r)
	state := scrobbler.StatePlaying
	if body.IsPaused {
		state = scrobbler.StatePaused
	}
	clientId, clientName := clientIdentity(ctx)
	err := api.scrobbler.ReportPlayback(ctx, scrobbler.ReportPlaybackParams{
		MediaId:      body.ItemId,
		PositionMs:   body.PositionTicks / 10_000,
		State:        state,
		PlaybackRate: 1.0,
		ClientId:     clientId,
		ClientName:   clientName,
	})
	if err != nil {
		log.Warn(ctx, "Jellyfin API: report playback progress failed", "id", body.ItemId, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// reportPlaybackStopped handles POST /Sessions/Playing/Stopped, sent once when
// playback of an item ends.
func (api *Router) reportPlaybackStopped(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body := decodeReport(r)
	clientId, clientName := clientIdentity(ctx)

	// IgnoreScrobble: Submit below is the sole source of the play-count increment
	// and external scrobble for this stop event. Without it, ReportPlayback's own
	// threshold-based scrobble logic would double-count the same play.
	err := api.scrobbler.ReportPlayback(ctx, scrobbler.ReportPlaybackParams{
		MediaId:        body.ItemId,
		PositionMs:     body.PositionTicks / 10_000,
		State:          scrobbler.StateStopped,
		IgnoreScrobble: true,
		ClientId:       clientId,
		ClientName:     clientName,
	})
	if err != nil {
		log.Warn(ctx, "Jellyfin API: report playback stopped failed", "id", body.ItemId, err)
	}

	err = api.scrobbler.Submit(ctx, []scrobbler.Submission{{TrackID: body.ItemId, Timestamp: time.Now()}})
	if err != nil {
		log.Warn(ctx, "Jellyfin API: scrobble failed", "id", body.ItemId, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// postCapabilities acknowledges Jellyfin session-capability negotiation.
// Navidrome doesn't track per-session client capabilities, so this is a no-op.
func (api *Router) postCapabilities(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
