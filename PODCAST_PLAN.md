# Podcast Support for navidrome-experimental

## Context

Navidrome (and this fork) has no podcast support today. The Subsonic API spec's podcast endpoints already exist as stubs returning HTTP 501 (`server/subsonic/api.go:229-230`: `getPodcasts`, `getNewestPodcasts`, `refreshPodcasts`, `createPodcastChannel`, `deletePodcastChannel`, `deletePodcastEpisode`, `downloadPodcastEpisode`).

The user wants podcasts usable both in the Navidrome web UI **and** through third-party Subsonic-protocol client apps ("Cirque" on Android and Windows) they already use to access their server. That second requirement is the constraint that drives most of the design: it's not enough to bolt a podcast list onto the web UI — episodes must be playable through the real `/rest/stream.view` endpoint that any Subsonic client calls, and the real spec endpoints must actually work.

The user also wants both playback modes available: stream-only (no local copy, like the existing Internet Radio feature) and download-and-store (episode cached to disk, works offline, needed for real Subsonic-client compatibility). Subscriptions are shared/admin-managed, mirroring how Internet Radio stations work today (not per-user).

**Reference implementation to mirror throughout:** the Internet Radio Station feature (`model/radio.go`, `persistence/radio_repository.go`, `db/migrations/20230115103212_create_internet_radio.go`, `server/nativeapi/radios.go`, `server/subsonic/radio.go`, `ui/src/radio/`) is the closest existing analog — an external-URL-based, admin-managed, non-scanned media resource — and the implementer should follow its patterns byte-for-byte wherever they apply.

---

## Architecture decision: how episodes stream through `/rest/stream.view`

`server/subsonic/stream.go`'s `Stream()` handler currently does one thing: `mf, err := api.ds.MediaFile(ctx).Get(id)`, then streams `*model.MediaFile` through `core/stream`. Podcast episodes are **not** MediaFile rows (we deliberately don't synthesize fake ones — `model.MediaFile` is deeply entangled with the scanner/library/folder subsystem, e.g. `AbsolutePath()` needs a real `library_id`/`folder_id`, and every read joins against the `library` table). Instead:

- Give podcast episodes their own model + repository (`model.PodcastEpisode`), no ID-prefix scheme needed — this codebase already has a precedent for polymorphic ID lookup: `model/get_entity.go`'s `GetEntityByID` tries each repository in turn until one succeeds (used today by `Download()`'s `model.GetEntityByID(ctx, api.ds, id)` call).
- In `Stream()`, try `ds.MediaFile(ctx).Get(id)` first (**zero change to the existing hot path** — no overhead for the 99.9% case of normal track streams). Only on `errors.Is(err, model.ErrNotFound)`, fall back to `ds.PodcastEpisode(ctx).Get(id)`. This is simpler and more consistent with the codebase's existing conventions than introducing ID prefixes.
- If the episode is downloaded (`DownloadStatus == Downloaded`), serve the local file directly via `http.ServeContent` (no FFmpeg transcode needed — podcast episodes are already-compressed audio; transcoding can be added later as a stretch goal).
- If not downloaded, behavior depends on the channel's `DownloadPolicy`:
  - `none` (stream-only mode): proxy the RSS enclosure URL through the server (`http.Get` + `io.Copy` into the response) so a Subsonic client never needs to know about the external URL — this is what makes stream-only mode actually usable by Cirque.
  - `new`/`all` (download-and-store mode) but not yet downloaded: trigger an on-demand download-then-serve with a bounded timeout, since some Subsonic clients call `stream.view` without first calling `downloadPodcastEpisode`.

This means the web player needs **no** `isRadio`-style raw-URL bypass for episodes — `subsonic.streamUrl(episode.id)` works universally, which is simpler than the Radio precedent.

`model/get_entity.go`'s `GetEntityByID` also gets two more tries added (`PodcastChannel`, `PodcastEpisode`) so `/rest/download.view` and any other polymorphic-ID code paths work too.

---

## Data model

### `model/podcast.go` (new)

Mirrors `model/radio.go`'s tag conventions (`structs:"..."` + `json:"..."`). Two structs:

**`PodcastChannel`** (parent, one per subscribed feed): `ID`, `Url` (feed URL, unique), `Title`, `Description`, `CoverArtUrl` (from feed's `<itunes:image>`), `UploadedImage` (admin override, like `Radio.UploadedImage`), `HomePageUrl`, `Status` (`new`/`downloading`/`completed`/`error` — mirrors the Subsonic spec's channel status vocabulary), `ErrorMessage`, `DownloadPolicy` (`none`/`new`/`all`), `RetentionCount`, `RetentionDays` (0 = unlimited), `LastCheckedAt`, `CreatedAt`, `UpdatedAt`, plus a transient `Episodes PodcastEpisodes` field (populated by `GetWithEpisodes`, mirroring `Playlist.Tracks`). `CoverArtID()` + `UploadedImagePath()` methods mirror `Radio`'s.

**`PodcastEpisode`** (child): `ID`, `ChannelID`, `Guid` (RSS `<guid>`, used for refresh dedup — fall back to enclosure URL if absent), `Title`, `Description`, `EnclosureUrl`, `ContentType`, `Size` (actual downloaded bytes, not the often-wrong advertised enclosure length), `Duration`, `PublishDate`, `DownloadStatus` (`not_downloaded`/`queued`/`downloading`/`downloaded`/`error`/`deleted`), `ErrorMessage`, `Path` (relative, under the podcasts storage folder), `Suffix`, `BitRate`, timestamps. `AbsolutePath()` mirrors `MediaFile.AbsolutePath()`, joining against `conf.Server.Podcasts.StorageFolder`.

Download-status state machine: `not_downloaded → queued → downloading → downloaded | error`; `downloaded → deleted` via retention/manual delete; `deleted`/`error → queued` on retry.

Repository interfaces (`PodcastChannelRepository`, `PodcastEpisodeRepository`) follow `RadioRepository`'s shape (`ResourceRepository` + `CountAll/Delete/Get/GetAll/Put`), plus channel-specific `GetWithEpisodes(id)` / `FindByUrl(url)` and episode-specific `FindByGuid(channelID, guid)` / `GetNewest(count)` (backs `getNewestPodcasts`).

### Supporting changes
- `model/datastore.go`: add `PodcastChannel(ctx) PodcastChannelRepository` and `PodcastEpisode(ctx) PodcastEpisodeRepository` to the `DataStore` interface.
- `consts/consts.go`: add `EntityPodcastChannel = "podcastChannel"` (mirrors `EntityRadio`), used by `UploadedImagePath`.
- `model/artwork_id.go`: add `KindPodcastChannelArtwork = Kind{"pc", "podcast_channel"}` to the `Kind` map (mirrors `KindRadioArtwork`), plus an `artworkIDFromPodcastChannel()` helper. Episodes reuse their parent channel's artwork (no separate per-episode art in Phase 1/2 — RSS episode-level images are a Phase 3 stretch goal).
- Migration `db/migrations/<next-timestamp>_create_podcasts.go` (check `db/migrations/` for the latest existing timestamp and bump past it): creates `podcast_channel` and `podcast_episode` tables (the latter with `channel_id references podcast_channel(id) on delete cascade`, `unique(channel_id, guid)`, and indexes on `channel_id`/`download_status`/`publish_date`). Structure mirrors `20230115103212_create_internet_radio.go`.

---

## Persistence layer

- `persistence/podcast_channel_repository.go` and `persistence/podcast_episode_repository.go` (new): mirror `persistence/radio_repository.go` structurally — embed `sqlRepository`, `r.registerModel(&model.PodcastChannel{}, map[string]filterFunc{...})`, admin-only `isPermitted()` for writes (reads open to any authenticated user, matching how `getInternetRadioStations` sits outside the admin-only route group). Implement `rest.Repository`/`rest.Persistable` for native REST reuse.
- `persistence/persistence.go`: add `PodcastChannel(ctx)`/`PodcastEpisode(ctx)` methods to `SQLStore`, plus `case model.PodcastChannel:`/`case model.PodcastEpisode:` in the `Resource(ctx, m any)` switch (same pattern as the existing `Radio` case).

---

## RSS refresh + download pipeline (`core/podcasts/`, new package)

- **Dependency:** add `github.com/mmcdole/gofeed` to `go.mod`. Real-world podcast feeds are inconsistent (missing GUIDs, six different date formats, `<itunes:duration>` as either `HH:MM:SS` or raw seconds, mixed RSS/Atom). Hand-rolling `encoding/xml` parsing means reimplementing gofeed's edge-case handling for no benefit — this is a case where the dependency clearly earns its place (same tier of justification as this project's existing use of `robfig/cron`).
- `core/podcasts/podcasts.go`: `Podcasts` interface + `podcasts` struct, constructed via `New(ds model.DataStore, broker events.Broker) Podcasts` (same constructor-injection style as other `core/` services). Methods: `CreateChannel`, `DeleteChannel`, `RefreshChannel`, `RefreshAll`, `DownloadEpisode`, `DeleteEpisode`, `RunRetention`.
- `core/podcasts/feed.go`: fetch + parse via `gofeed.NewParser().ParseURLWithContext`, map `gofeed.Item` → `model.PodcastEpisode` (GUID fallback, duration parsing, enclosure extraction).
- `core/podcasts/refresh.go`: `RefreshChannel` sets `Status=downloading`, fetches feed, upserts episodes by `FindByGuid` (update in place if exists, insert if new), sets `Status=completed`/`error` + `LastCheckedAt`. If `DownloadPolicy != none`, enqueues new episodes (and all existing `not_downloaded` ones if policy is `all`) onto the download pipeline. Emits SSE events before/after (see below).
- `core/podcasts/download.go`: concurrent download pipeline using the **same `go-pipeline` (`ppl`) library the scanner already uses** (`scanner/phase_1_folders.go:200-205` is the template) — `ppl.NewStage(downloadOne, ppl.Concurrency(conf.Server.Podcasts.DownloadConcurrency))`. Each download: `http.Get` with context timeout, stream to a temp file, derive suffix from content-type, atomic rename, update the episode row with real `Path`/`Suffix`/`Size`, set `DownloadStatus=Downloaded`. Errors set `DownloadStatus=error` + `ErrorMessage`, clean up the temp file.
- `core/podcasts/naming.go`: files live at `{StorageFolder}/{channelID}/{episodeID}.{suffix}` — using DB IDs rather than sanitized RSS titles avoids collisions/path-traversal entirely; display names always come from the DB `Title` field. Directory-per-channel makes channel deletion a single `os.RemoveAll`.
- `core/podcasts/retention.go`: `RunRetention` enforces `RetentionCount`/`RetentionDays` per channel — deletes local files beyond the policy and sets those episodes to `DownloadStatus=deleted` (row kept, `Path` cleared).
- Wire this into `core/wire_providers.go`'s existing `Set` directly (same place `playlists.NewPlaylists` is registered, per `core/wire_providers.go:23`) — add `podcasts.New`.

### Scheduling
- `cmd/root.go`: add `schedulePodcastRefresh(ctx)`, copying `schedulePeriodicScan`'s shape (lines ~143-167) — reads `conf.Server.Podcasts.Schedule`, calls `scheduler.GetInstance().Add(schedule, func() { podcastsService.RefreshAll(ctx) })`. Registered via `g.Go(schedulePodcastRefresh(ctx))` next to the existing scan scheduling call. Add `CreatePodcastsService(ctx)` to `cmd/wire_injectors.go` (mirrors `CreateScanner`).
- `conf/configuration.go`: new `podcastsOptions` struct (`Enabled`, `StorageFolder Dir`, `Schedule`, `DownloadConcurrency`, `DefaultDownloadPolicy`, `MaxDownloadSizeMB`), added to `configOptions` as `Podcasts podcastsOptions`. Default-derivation for `StorageFolder` mirrors the existing `CacheFolder` block (`DataFolder/podcasts` if unset). `validatePodcastSchedule` added to the config validation chain, mirroring `validateScanSchedule`.

### Progress events
- `server/events/events.go`: add `PodcastRefreshStatus` and `PodcastDownloadStatus` events (same `baseEvent`-embedding shape as the existing `ScanStatus`). `core/podcasts` broadcasts via `broker.SendBroadcastMessage` at each state transition, plus `events.RefreshResource{}` after mutations so React-admin list views auto-refresh (existing generic mechanism).

---

## Subsonic API surface (`server/subsonic/podcast.go`, new)

Replace the `h501(r, "getPodcasts", ...)` line at `server/subsonic/api.go:229-230` with real registrations, gated behind `conf.Server.Podcasts.Enabled` the same way `EnableSharing` gates the share endpoints (`api.go:207-217` — `if enabled { h(...) } else { h501(...) }`):

```go
if conf.Server.Podcasts.Enabled {
    r.Group(func(r chi.Router) {
        r.Use(getPlayer(api.players))
        h(r, "getPodcasts", api.GetPodcasts)
        h(r, "getNewestPodcasts", api.GetNewestPodcasts)
        r.Group(func(r chi.Router) {
            r.Use(adminOnly)
            h(r, "refreshPodcasts", api.RefreshPodcasts)
            h(r, "createPodcastChannel", api.CreatePodcastChannel)
            h(r, "deletePodcastChannel", api.DeletePodcastChannel)
            h(r, "deletePodcastEpisode", api.DeletePodcastEpisode)
            h(r, "downloadPodcastEpisode", api.DownloadPodcastEpisode)
        })
    })
} else {
    h501(r, "getPodcasts", "getNewestPodcasts", "refreshPodcasts", "createPodcastChannel",
        "deletePodcastChannel", "deletePodcastEpisode", "downloadPodcastEpisode")
}
```

Handler notes (matching real Subsonic spec semantics):
- `CreatePodcastChannel`: param `url` only (spec derives title from the feed itself) — dedup via `FindByUrl`, insert, then a first `RefreshChannel`.
- `RefreshPodcasts`/`DownloadPodcastEpisode`: **fire-and-forget** per spec — kick a goroutine with a detached context, return immediately.
- `DeletePodcastChannel`/`DeletePodcastEpisode`: synchronous (fast: DB row + local files).
- `GetPodcasts`: params `includeEpisodes` (bool, default true), optional `id` (single channel).
- `GetNewestPodcasts`: param `count` (default 20), backed by `PodcastEpisodeRepository.GetNewest`.

New response structs in `server/subsonic/responses/responses.go` (near the existing `InternetRadioStations` struct): `Podcasts{Channel []PodcastChannel}`, `PodcastChannel{ID, Url, Title, Description, CoverArt, OriginalImageUrl, Status, ErrorMessage, Episode []PodcastEpisode}`, `PodcastEpisode` (embeds the existing `Child` struct like a song does, plus `StreamId`, `ChannelId`, `Description`, `Status`, `PublishDate`), `NewestPodcasts{Episode []PodcastEpisode}`. Key field-mapping details:
- `episode.id` = the podcast episode's own DB ID (what clients pass back into `stream.view`/`download.view`).
- `episode.streamId` = same ID, populated **regardless of download state** (since `stream.go`'s proxy fallback makes every episode streamable) — this is what makes stream-only mode actually work for Subsonic clients.
- `episode.status`: map `DownloadStatus` → spec vocabulary (`not_downloaded`→`skipped`/`new` depending on policy, `queued`/`downloading`→`downloading`, `downloaded`→`completed`, `error`→`error`, `deleted`→`deleted`).
- `episode.parent`/`album`/`artist` = the channel's ID/title (common convention other Subsonic servers use, so Cirque's UI renders sensibly).
- `episode.coverArt` = the channel's `CoverArtID().String()`.

---

## Native REST API (`server/nativeapi/podcasts.go`, new)

Mirrors `server/nativeapi/radios.go`'s route structure, with one deviation: unlike Radio (a pure data record), channel/episode creation and deletion have side effects (feed fetch, file cleanup). Rather than coupling `persistence/` to `core/podcasts` (architecturally inconsistent with this codebase's layering), add thin custom handlers that call `core/podcasts.Podcasts` methods directly for the side-effecting actions (`create`, `delete`, `refresh`, `download`), while still using generic `rest.GetAll`/`rest.Get`/`rest.Put` for plain reads/updates:

- `POST /api/rest/podcastChannel` → custom handler calling `podcasts.CreateChannel`
- `GET/PUT /api/rest/podcastChannel/{id}` → generic `rest.Get`/`rest.Put`
- `DELETE /api/rest/podcastChannel/{id}` → custom handler calling `podcasts.DeleteChannel`
- `POST /api/rest/podcastChannel/{id}/refresh` → custom handler calling `podcasts.RefreshChannel`
- `POST/DELETE /api/rest/podcastChannel/{id}/image` → mirrors `uploadRadioImage`/`deleteRadioImage` verbatim
- `GET /api/rest/podcastEpisode`, `GET /api/rest/podcastEpisode/{id}` → generic `rest.GetAll`/`rest.Get`
- `POST /api/rest/podcastEpisode/{id}/download` → custom handler calling `podcasts.DownloadEpisode`
- `DELETE /api/rest/podcastEpisode/{id}` → custom handler calling `podcasts.DeleteEpisode`

Register via `api.addPodcastRoutes(r)` in `server/nativeapi/native_api.go`, alongside `api.addRadioRoute(r)`.

---

## Web UI (`ui/src/podcast/`, new)

Mirrors `ui/src/radio/` (`index.jsx`, `*List.jsx`, `*Create.jsx`, `*Edit.jsx`, `helper.jsx`):

- `PodcastChannelList.jsx`: channel title/episode count/status/download policy, admin-only delete/refresh actions, row click drills into that channel's episodes.
- `PodcastChannelCreate.jsx`: single `url` field (spec derives everything else from the feed).
- `PodcastChannelEdit.jsx`: edit `downloadPolicy`/`retentionCount`/`retentionDays` + image upload (mirrors `RadioEdit.jsx`).
- `PodcastEpisodeList.jsx` (new concept, no direct Radio analog): episodes scoped to a channel, showing title/publish date/duration/download-status badge, with download/delete (admin) and play actions.
- `helper.jsx`: `songFromPodcastEpisode(episode, channel)` — simpler than `songFromRadio()` since (per the streaming design above) `subsonic.streamUrl(episode.id)` works universally; no `isRadio`-style raw-URL bypass needed in `playerReducer.js`.
- `ui/src/App.jsx`: `import podcast from './podcast'`, then `<Resource name="podcastChannel" {...(permissions === 'admin' ? podcast.admin : podcast.all)} />` and `<Resource name="podcastEpisode" {...podcastEpisode} />` (episodes aren't a separate sidebar item — same pattern as playlist tracks). **No `ui/src/layout/Menu.jsx` edit needed** — the sidebar is auto-generated from registered `<Resource>` elements with an `icon` prop (confirmed: Radio's sidebar entry comes entirely from `radio/index.jsx`'s `DynamicMenuIcon`, not a Menu.jsx edit).

---

## Phased roadmap

**Phase 1 — Data model, RSS refresh, native REST, minimal web UI (web-only, stream-only playback)**
Model + migration + persistence + `core/podcasts` (refresh only, no download pipeline) + gofeed dependency + config (`Enabled`/`Schedule`/`DefaultDownloadPolicy`) + scheduler wiring + artwork reader + native REST (channel CRUD + refresh) + web UI (channel list/create/edit, read-only episode list). Episodes play via a temporary direct-URL player bypass (copy of the `isRadio` pattern) since `stream.go` doesn't branch yet. **Outcome:** admin can subscribe to a feed and play episodes in the Navidrome web UI. **Status: done.**

**Phase 1.5 — Podcast discovery (search + live regional top charts)**
Requiring users to already have an RSS feed URL is a poor onboarding experience, so this phase adds real discovery before Phase 2's heavier download-pipeline work:
- `core/podcasts` gains `SearchFeeds(ctx, query string) ([]FeedSearchResult, error)`, calling the free, keyless iTunes Search API (`https://itunes.apple.com/search?media=podcast&entity=podcast&term=...`). Confirmed response fields: `collectionName` (title), `artistName` (author), `feedUrl`, `artworkUrl600`/`artworkUrl100`, `trackCount` (episode count), `collectionId`.
- `core/podcasts` also gains `TopFeeds(ctx, country string) ([]FeedSearchResult, error)`, backed by Apple's public Marketing Tools chart API (`https://rss.marketingtools.apple.com/api/v2/{country}/podcasts/top/{n}/podcasts.json`) for a genuinely region-specific "top podcasts" list (e.g. AU and US charts return completely different, real shows), with a follow-up bulk `https://itunes.apple.com/lookup?id=...` call to resolve each chart entry's actual RSS feed URL (in chart order). Invalid/unrecognized country codes fall back to the US chart. Both confirmed working end-to-end against live requests.
- New native REST endpoints `GET /api/rest/podcastChannel/search?q=...` and `GET /api/rest/podcastChannel/top?country=XX` (admin-only, matching create/delete) proxying both server-side so the browser never calls Apple directly (avoids CORS, keeps the external dependency server-side).
- `PodcastChannelCreate.jsx` becomes search-first: a search box returning result cards (artwork, title, author, episode count) with a one-click "Subscribe" button that calls the existing `createPodcastChannel` flow with the result's `feedUrl`; manual URL entry kept as a fallback/advanced option.
- A shared `TopFeedsSuggestions` component (country guessed client-side from `navigator.language`, no IP lookups) renders the live regional top-chart as one-click quick-add chips. It's used in **two** places — the empty-state Podcasts list (first-run) *and* always-visible on the Create/search page itself — since a first version that only showed suggestions on the empty list became unreachable the moment a user subscribed to anything.
**Outcome:** subscribing to a podcast is "search by name" or "pick a trending show for your region", always reachable via the Create page, not just requiring a raw feed URL on first run.

**Phase 2 — Subsonic API + download-to-disk + real stream.view serving (unlocks Cirque)**
Finish config (`StorageFolder`/`DownloadConcurrency`/`MaxDownloadSizeMB`) + download pipeline (`go-pipeline`) + SSE progress events + all 7 Subsonic handlers + response structs + the `stream.go`/`Download()` branching logic + native REST download/delete actions + web UI download-status badges, and remove the Phase 1 player bypass now that `stream.view` handles every episode. **Outcome:** the user's actual goal — Cirque (Android/Windows) can subscribe, browse, download, and stream podcasts exactly like any other Subsonic server. **Status: done.**

**Phase 3 — Retention enforcement, playlist integration, personal UX polish**
**Status: done.** Scoped from a design discussion covering three linked pieces:

1. **Retention enforcement** (`core/podcasts/retention.go`, new) — `RetentionCount`/`RetentionDays` already exist as columns on `PodcastChannel` (added in Phase 1) but are currently dormant; nothing reads or acts on them yet. This phase adds the actual `RunRetention(ctx)` enforcement, run per-channel after each scheduled refresh (same job as `schedulePodcastRefresh`) and available as an explicit trigger:
   - Enforces `RetentionCount` (max episodes kept) and `RetentionDays` (max age), oldest-downloaded-first, same semantics as `DeleteEpisode` (file removed, row kept, `DownloadStatus=deleted`, `Path`/`Suffix`/`Size` cleared).
   - Decision: **no separate "number of episodes to download" / "max downloads" settings** — collapsed into `RetentionCount`/`RetentionDays`, since a distinct field would just be a second name for the same "cap how many stay downloaded" behavior. Paired with the existing `DownloadPolicy` (`none`/`new`/`all`), e.g. policy `all` + `RetentionCount: 10` already means "download everything as it publishes, keep only the newest 10 on disk" with no extra knob needed.
   - New field: per-channel `MaxStorageMB` (0 = unlimited) — a genuinely distinct axis from count/age, since episode file size varies hugely by show (a 20-minute news podcast vs. a 2-hour video-heavy interview), so a count-based cap alone doesn't bound disk usage predictably. Enforced the same oldest-first way when exceeded.
   - Retention must skip any episode currently referenced by a playlist (see pinning rule below) — being in a playlist counts as "in use."

2. **Playlist integration, downloaded episodes only** — the actual feature request. Podcast episodes may be added to regular playlists, but **only once downloaded** (`DownloadStatus == Downloaded`); stream-only/not-yet-downloaded episodes are rejected at add-time (both native REST and Subsonic `createPlaylist`/`updatePlaylist`). This restriction is what keeps the feature simple:
   - Schema: relax `playlist_track` to reference an episode as well as a `MediaFile`, reusing the polymorphic-ID pattern already established by `model.GetEntityByID` and `stream.go`'s MediaFile-then-PodcastEpisode fallback, rather than a full playlist rearchitecture.
   - Drag-to-reorder comes for free — it's the existing playlist position mechanism, agnostic to what the underlying entity is.
   - M3U export (`model/mediafile.go`'s `ToM3U8`, also used by playlist share links) writes local filesystem paths per line, with no concept of "stream from the server." Because only *downloaded* episodes can ever be in a playlist, every entry always has a real file on disk, so no special-casing is needed for the stream-only case. Keep a defensive fallback (skip the row if `Path` is unexpectedly empty) rather than assuming a file can never go missing.
   - **Cascade rule**: deleting an episode's download — via manual delete *or* via retention cleanup above — also removes it from any playlist it belongs to, not just clears the file. This is what prevents dangling/broken playlist entries.
   - **Retention pinning**: an episode referenced by at least one playlist is exempt from automatic retention cleanup (implicitly "in use," shouldn't be silently deleted out from under a queued playlist). Manual delete still works and cascades per the rule above.

3. **Personal UI toggle** — a per-user, per-browser preference controlling whether the Podcasts sidebar entry shows up at all, mirroring the existing Folders toggle exactly (`ui/src/personal/FolderViewToggle.jsx`, `state.settings.showFolderView`, Redux `settings` slice + Personal Options panel). **Status: done** — `ui/src/personal/PodcastsToggle.jsx` (`state.settings.showPodcasts`) hides/shows the `podcastChannel` resource's auto-generated sidebar link in `ui/src/layout/Menu.jsx`, same declutter-only semantics as Folders (not a capability gate — that's the existing server-wide `conf.Server.Podcasts.Enabled`).

4. **"Listened" indicator** (added after the above three, same phase in spirit) — episodes participate in the same per-user `annotation` table songs use (`play_count`/`play_date`), scoped to just play-tracking (no starring/rating - that needs an `average_rating` column podcast_episode doesn't have). `persistence/podcast_episode_repository.go`'s `withPlayAnnotation` left-joins it into `Get`/`GetAll`/`GetNewest`. `server/subsonic/podcast_stream.go`'s `streamPodcastEpisode` records a play as soon as a client starts streaming (no position/completion threshold - see Phase 4 item 1 for why that's a separate, bigger piece). Web UI shows a checkmark column in the episode list. **Status: done.**

Carried over from the original Phase 3 scope, still applicable: live SSE-driven download progress bars in the UI, optional per-episode artwork, optional transcoding support for podcast streams, optional Range-header passthrough on the stream-only proxy path (for seeking), docs page.

---

**Phase 4 — Resume position, continue-listening queue, OPML import/export**
Not started. Scoped from a discussion about what would most improve the actual day-to-day podcast experience (mostly used via Cirque, a third-party Subsonic client, not just the web UI) rather than from a specific user request:

1. **Resume position ("in progress" tracking)** — the biggest usability gap for podcasts specifically: a 90-minute episode isn't finished in one sitting, and today every replay starts from zero. There's no existing per-track resume mechanism anywhere in Navidrome to reuse (`PlayQueue` is a single per-user "current item + position" snapshot backing Subsonic's `savePlayQueue`/resume, not a historical position kept per item across a whole library) - this is genuinely new capability, not a "extend an existing thing" job like the listened indicator was.
   - Schema: add `play_position` (integer, milliseconds) to `annotation` - the correct per-user, per-item table, alongside the `play_count`/`play_date` this phase already uses for episodes. Reusing `annotation` rather than a `podcast_episode` column matters: playback position is inherently per-*user*, and podcast subscriptions/episodes are shared/admin-managed across the whole server, so a column on `podcast_episode` itself would incorrectly share one position across every user.
   - `model/annotation.go`: add `PlayPosition int64` to `Annotations`. `persistence/sql_annotations.go`: add a generic `SetPlayPosition(itemID string, positionMs int64) error` alongside the existing `IncPlayCount` (safe to add to the shared `sqlRepository` mixin the same way - doesn't touch `average_rating`, so no repeat of the `SetRating` footgun noted in Phase 3 item 4).
   - Wire-up point: `server/subsonic/media_annotation.go`'s `ReportPlayback()` already receives a `mediaType` param but only logs a warning and ignores it for anything other than `"song"` - this is where episode-aware position tracking plugs in, dispatching to `PodcastEpisode(ctx).SetPlayPosition(...)` instead of the MediaFile-only `PlayTracker.ReportPlayback` when the id resolves to an episode.
   - Auto-clear position back to 0 once playback crosses ~95% of the episode's duration (treat as "finished," matching standard podcast-app behavior - otherwise a listener who reaches the very end forever gets offered a "resume near the end" prompt instead of a fresh replay).
   - **Open question, needs checking before implementation**: this repo already advertises the OpenSubsonic `playbackReport` extension (`getOpenSubsonicExtensions`) - verify whether that extension's spec already defines a standard way for a *client* to read back a stored resume position (vs. just report one), so this aligns with the extension rather than inventing a parallel, Navidrome-only field third-party clients like Cirque have no reason to read.
   - `persistence/podcast_episode_repository.go`'s `withPlayAnnotation` gains `play_position`, `model.PodcastEpisode` gains `PlayPosition int64`. Web UI surfaces "resume from mm:ss" wherever an episode is shown/played.

2. **Cross-channel "Up Next" / continue-listening queue** — today, episodes are only browsable one channel at a time (`ReferenceManyField target="channel_id"` in `PodcastChannelShow.jsx`). Every mainstream podcast app's home screen is a single unified feed of newest-unplayed episodes across every subscription - low marginal backend cost given `GetNewest` and the listened-tracking from Phase 3 item 4 already exist, but a materially different daily workflow (check one feed of what's new, not N channel pages).
   - New repository method, e.g. `GetUnlistened(count int) (PodcastEpisodes, error)` - LEFT JOIN `annotation` scoped to the current user, filtering `coalesce(play_count, 0) = 0`, ordered by `publish_date desc`. Distinct from `GetNewest` (which has no listened-filter) and needs its own query rather than a Go-side filter, since it should scan efficiently across every channel server-wide.
   - Deliberately **not** added to the spec-compliant `getNewestPodcasts` Subsonic endpoint (real `getNewestPodcasts` has no "unlistened" concept in the spec) - keep that endpoint spec-compliant/unchanged, and add this as a native-only endpoint + web UI view, since it's a Navidrome-experimental value-add rather than a protocol feature.
   - No download-status filter - a stream-only episode is just as eligible as a downloaded one, since `stream.view` already resolves either transparently (Phase 2).
   - UI: a new cross-channel list/section (e.g. replacing or supplementing the current empty-state suggestions area), each row showing its channel name/artwork since channel context isn't implicit here the way it is inside a single channel's episode list.

3. **OPML import/export** — the standard interchange format every podcast app (Overcast, Pocket Casts, AntennaPod, etc.) supports for subscription lists. Right now the only way in or out of this server's subscription list is one feed at a time through search/manual-URL entry - OPML lets someone migrate an existing list wholesale, or back theirs up independent of the database.
   - `core/podcasts` gains `ImportOPML(ctx, reader io.Reader) (imported, skipped int, errs []error)` - parse with stdlib `encoding/xml` (OPML is simple enough not to need a new dependency, unlike gofeed for RSS), recursively walk `<outline>` elements for `xmlUrl` attributes, call the existing `CreateChannel` per feed (already dedups via `FindByUrl`) - one bad feed in the file shouldn't abort the whole import.
   - `core/podcasts` gains `ExportOPML(ctx) ([]byte, error)` - builds an OPML document (title + url per subscribed channel) via `encoding/xml` marshaling.
   - New native REST endpoints, admin-only (matching `createPodcastChannel`'s existing permission convention): `POST /api/rest/podcastChannel/importOpml` (file upload), `GET /api/rest/podcastChannel/exportOpml` (file download, `Content-Disposition: attachment`, mirroring the existing playlist M3U export convention).
   - UI: "Import OPML" (file picker dialog) and "Export OPML" (download trigger) actions on the Podcasts list page, admin-only, alongside the existing search-first subscribe flow.

---

## Critical files

- `model/podcast.go` (new) — data model, download-status state machine
- `persistence/podcast_channel_repository.go`, `persistence/podcast_episode_repository.go` (new) — mirror `persistence/radio_repository.go`
- `db/migrations/<ts>_create_podcasts.go` (new) — schema
- `core/podcasts/refresh.go`, `core/podcasts/download.go` (new) — gofeed-based refresh, go-pipeline-based download
- `server/subsonic/stream.go` (modify) — MediaFile-first, PodcastEpisode-fallback branch in `Stream()`; `PodcastEpisode` case in `Download()`
- `server/subsonic/podcast.go` (new) — the 7 spec endpoints
- `server/subsonic/responses/responses.go` (modify) — `Podcasts`/`PodcastChannel`/`PodcastEpisode` response structs
- `server/nativeapi/podcasts.go` (new) — native REST, mirrors `radios.go`
- `ui/src/podcast/` (new) — mirrors `ui/src/radio/`
- `core/podcasts/retention.go` (Phase 3, new) — `RunRetention`, count/days/size-based cleanup, playlist-pinning exemption
- `model/playlist.go`, `persistence/playlist_repository.go`, migration (Phase 3, modify) — polymorphic playlist entries (MediaFile or PodcastEpisode)
- `model/mediafile.go`'s `ToM3U8` (Phase 3, modify) — defensive empty-`Path` fallback
- `ui/src/personal/` (Phase 3, new toggle) — personal preference mirroring `FolderViewToggle.jsx`
- `persistence/podcast_episode_repository.go`'s `withPlayAnnotation` (Phase 3 item 4, new) — listened-tracking join
- `model/annotation.go`, `persistence/sql_annotations.go` (Phase 4, modify) — `PlayPosition`/`SetPlayPosition`
- `server/subsonic/media_annotation.go`'s `ReportPlayback` (Phase 4, modify) — episode-aware position dispatch
- `core/podcasts/` (Phase 4, new) — `GetUnlistened` query, `ImportOPML`/`ExportOPML`
- `server/nativeapi/podcasts.go` (Phase 4, modify) — `importOpml`/`exportOpml`/unlistened-queue routes

## Verification

- Backend: `go test ./model/... ./persistence/... ./core/podcasts/... ./server/subsonic/... ./server/nativeapi/...` plus existing `make test` suite for regressions.
- Manual: subscribe to a real RSS feed via the web UI, confirm episodes appear and play (Phase 1); then via a Subsonic API client — ideally the user's actual Cirque app — confirm `getPodcasts`, `createPodcastChannel`, `downloadPodcastEpisode`, and streaming a downloaded + a stream-only episode all work end-to-end (Phase 2).
- `npm run lint`/`npm test`/`npm run build` in `ui/` for the frontend pieces.
