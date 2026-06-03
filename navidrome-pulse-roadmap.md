# Navidrome Enhanced Scrobbling & Pulse Integration - Roadmap

This document outlines the steps to enhance Navidrome's scrobbling engine and Plugin API (PDK) to support deep listening analytics for the Pulse plugin and other clients.

---

## Phase 1: Data Model & Persistence

### 1. Database Migration
*   **Action**: Create a new migration to add attribution columns to the `scrobble` table.
*   **Fields**: `client` (string), `source` (string), `origin` (string), `playback_mode` (string).
*   **Status**: ⏳ Pending

### 2. Core Model Update
*   **Action**: Update the internal `Scrobble` struct in `model/scrobble.go` (or relevant location) to include the new attribution fields.
*   **Status**: ⏳ Pending

---

## Phase 2: Plugin API (PDK) Enhancements

### 3. Update PDK Interface
*   **Action**: Modify `plugins/capabilities/scrobbler.go` to include `Source`, `Origin`, and `PlaybackMode` in the `ScrobbleRequest`.
*   **Status**: ⏳ Pending

### 4. Update Plugin Adapter
*   **Action**: Update `plugins/scrobbler_adapter.go` to extract attribution data from Navidrome's context/models and pass it to the `.wasm` plugin.
*   **Status**: ⏳ Pending

---

## Phase 3: API & Engine Integration

### 5. Subsonic API Update (`scrobble.view`)
*   **Action**: Update `server/subsonic/media_annotation.go` to accept optional `source`, `origin`, and `playback_mode` parameters.
*   **Status**: ⏳ Pending

### 6. Play Tracker Logic
*   **Action**: Update `core/scrobbler/play_tracker.go` to capture the `ClientName` (from Subsonic context) and the new optional parameters, ensuring they are saved to the DB and dispatched to plugins.
*   **Status**: ⏳ Pending

---

## Goal: Native Pulse Integration
*   ✅ Eliminate the need for the external `pulse-bridge`.
*   ✅ Allow the Pulse plugin to automatically detect "Android Auto" vs "Web" vs "Windows Desktop".
*   ✅ Enable "Top Origin" stats (e.g., "You mostly listen to your 'Favorites' mix").
