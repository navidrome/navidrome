# Navidrome Enhanced Scrobbling & Pulse Integration - Roadmap

This document outlines the steps to enhance Navidrome's scrobbling engine and Plugin API (PDK) to support deep listening analytics for the Pulse plugin and other clients.

---

## Phase 1: Data Model & Persistence

### 1. Database Migration
*   **Action**: Create a new migration to add attribution columns to the `scrobble` table.
*   **Fields**: `client` (string), `source` (string), `origin` (string), `playback_mode` (string).
*   **Status**: ✅ Completed

### 2. Core Model Update
*   **Action**: Update the internal `Scrobble` struct in `model/scrobble.go` (or relevant location) to include the new attribution fields.
*   **Status**: ✅ Completed

---

## Phase 2: Plugin API (PDK) Enhancements

### 3. Update PDK Interface
*   **Action**: Modify `plugins/capabilities/scrobbler.go` to include `Source`, `Origin`, and `PlaybackMode` in the `ScrobbleRequest`.
*   **Status**: ✅ Completed

### 4. Update Plugin Adapter
*   **Action**: Update `plugins/scrobbler_adapter.go` to extract attribution data from Navidrome's context/models and pass it to the `.wasm` plugin.
*   **Status**: ✅ Completed

---

## Phase 3: API & Engine Integration

### 5. Subsonic API Update (`scrobble.view`)
*   **Action**: Update `server/subsonic/media_annotation.go` to accept optional `source`, `origin`, and `playback_mode` parameters.
*   **Status**: ✅ Completed

### 6. Play Tracker Logic
*   **Action**: Update `core/scrobbler/play_tracker.go` to capture the `ClientName` (from Subsonic context) and the new optional parameters, ensuring they are saved to the DB and dispatched to plugins.
*   **Status**: ✅ Completed

---

## Goal: Native Pulse Integration
*   ✅ Eliminate the need for the external `pulse-bridge`.
*   ✅ Allow the Pulse plugin to automatically detect "Android Auto" vs "Web" vs "Windows Desktop".
*   ✅ Enable "Top Origin" stats (e.g., "You mostly listen to your 'Favorites' mix").

---

## Phase 4: Podcast Attribution & Plugin-System Hardening (2026-07-29)

Follow-up work after Phase 1-3 shipped, triggered by a real incident: Pulse showed as enabled with zero data after
an update, traced to two stacked, independent bugs in the plugin system (not in Pulse's own code) plus two feature
gaps found while fixing them.

### 7. `nd_source`: validated device-type attribution
*   **Action**: Add a stricter, allowlist-validated `nd_source` query param (`android_phone`, `android_tablet`,
    `android_tv`, `android_auto`, `windows_desktop`) alongside the original free-form `source` field, read on both
    `scrobble.view` and podcast episode streaming. Invalid/missing values clear to empty rather than rejecting the
    request - standard Subsonic clients that never send it are unaffected.
*   **Status**: ✅ Completed (`server/subsonic/nd_source.go`, `media_annotation.go`, `podcast_stream.go`)

### 8. Podcast plays now dispatch to plugins too
*   **Action**: New `PodcastScrobbler` capability (`OnPodcastPlayed`/`PodcastPlayedRequest`, mirroring `Scrobbler`)
    so podcast episode plays fire to plugins the same way song scrobbles already did. Before this, Pulse (and any
    other listening-stats plugin) only ever saw song plays - podcast listening was invisible to it entirely.
*   **Status**: ✅ Completed (`plugins/capabilities/podcast_scrobbler.go`, `plugins/podcast_scrobbler_adapter.go`)

### 9. Go PDK bindings were silently dropping attribution fields
*   **Found while regenerating bindings for item 7/8 above**: `plugins/pdk/go/scrobbler`'s generated Go struct -
    the one Go plugins actually compile against - was missing the `Client`/`Source`/`Origin`/`PlaybackMode` fields
    entirely, even though the host side (`plugins/capabilities/scrobbler.go`, `plugins/scrobbler_adapter.go`) had
    already been sending them in every scrobble's JSON payload. Since Go's `encoding/json` silently drops unknown
    fields on unmarshal, this meant **Go plugins could never actually read scrobble attribution**, regardless of
    what the Subsonic client sent - a real, previously-undiagnosed gap between what looked complete on the host
    side and what plugins could actually receive.
*   **Status**: ✅ Fixed by regenerating PDK bindings via `ndpgen` (`make gen`-equivalent) - no manual edits, since
    those files are generated from `plugins/capabilities/*.go`.

### 10. Plugin default timeout: 30s -> 5min for data-processing plugins
*   **Action**: `plugins/manager.go`'s `defaultTimeout` was 30 seconds, tuned for typical request/response plugin
    calls - too short for Pulse's `runUpdate` (reads every KVStore play-log entry, computes weekly/monthly/yearly
    stats, writes a full snapshot to a playlist comment). A run that didn't finish in 30s was silently killed
    mid-write, which is exactly the kind of failure that looks like "no data" from the outside.
*   **Status**: ✅ Completed - `defaultTimeout` raised to 5 minutes for this class of plugin.

### 11. (Separate, server-side) Plugin schema-resync bug that force-disabled Pulse
*   **Root cause of the actual "enabled but no data" incident**, not a Pulse bug at all: `updatePluginInDB`
    (`plugins/manager_sync.go`) unconditionally forced `Enabled = false` any time a plugin's manifest schema
    version was re-stamped, even when the underlying `.ndp` file hadn't actually changed. A routine schema-version
    bump on the server silently disabled Pulse (and any other already-enabled plugin) without any error surfaced
    to the admin.
*   **Status**: ✅ Fixed - schema-only re-extractions of an already-enabled plugin now transparently reload and
    stay enabled, falling back to disabled-with-`LastError` only if the reload itself actually fails. Real file
    changes still force a manual re-enable, unchanged. See PR #30 (`fix/plugin-schema-resync-preserves-enabled`).

**Status of items 7-10 relative to a tagged release:** develop-only as of this writing (unreleased since
`v0.63.2-experimental.5`). Item 11 lives on a separate, still-open PR.
