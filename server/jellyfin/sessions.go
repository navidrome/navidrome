package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"

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

// decodeReport reads the playback report body. ItemId falls back to a query param (some clients send
// it there) and is decoded here since it flows straight into scrobbler lookups by media file id.
// Finamp reports restored-queue playback with truncated ids, hence resolveItemID.
func (api *Router) decodeReport(r *http.Request) playbackReport {
	var body playbackReport
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.ItemId == "" {
		body.ItemId = r.URL.Query().Get("itemid")
	}
	body.ItemId = api.resolveItemID(r.Context(), dto.DecodeID(body.ItemId))
	return body
}

// clientIdentity returns the scrobbler cache key/display name for the caller's
// player. Both are zero values if withPlayer could not resolve a player.
func clientIdentity(ctx context.Context) (id, name string) {
	player, _ := request.PlayerFrom(ctx)
	return player.ID, player.Client
}

// reportPlaybackStart handles POST /Sessions/Playing, sent once when a client starts an item.
//
// These Sessions endpoints report only the caller's own playback and never expose content, so unlike
// browse/stream they are intentionally not library-access-gated.
func (api *Router) reportPlaybackStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body := api.decodeReport(r)
	clientId, clientName := clientIdentity(ctx)
	err := api.scrobbler.ReportPlayback(ctx, scrobbler.ReportPlaybackParams{
		MediaId:      body.ItemId,
		PositionMs:   dto.MillisFromTicks(body.PositionTicks),
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
	body := api.decodeReport(r)
	state := scrobbler.StatePlaying
	if body.IsPaused {
		state = scrobbler.StatePaused
	}
	clientId, clientName := clientIdentity(ctx)
	err := api.scrobbler.ReportPlayback(ctx, scrobbler.ReportPlaybackParams{
		MediaId:      body.ItemId,
		PositionMs:   dto.MillisFromTicks(body.PositionTicks),
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

// reportPlaybackStopped handles POST /Sessions/Playing/Stopped, sent once when playback ends.
//
// Jellyfin clients (Finamp) send a Stopped report on *every* stop, even an immediate track switch,
// so the play threshold is applied server-side: ReportPlayback's StateStopped logic counts the play
// only past 50% (capped at 4 minutes). Force-submitting here would mark a one-second skip as played.
func (api *Router) reportPlaybackStopped(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body := api.decodeReport(r)
	clientId, clientName := clientIdentity(ctx)

	err := api.scrobbler.ReportPlayback(ctx, scrobbler.ReportPlaybackParams{
		MediaId:    body.ItemId,
		PositionMs: dto.MillisFromTicks(body.PositionTicks),
		State:      scrobbler.StateStopped,
		ClientId:   clientId,
		ClientName: clientName,
	})
	if err != nil {
		log.Warn(ctx, "Jellyfin API: report playback stopped failed", "id", body.ItemId, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// postCapabilities acknowledges Jellyfin session-capability negotiation.
// Navidrome doesn't track per-session client capabilities, so this is a no-op.
func (api *Router) postCapabilities(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
