# Feature Roadmap — Community-Requested Ideas

This tracks feasibility assessments and build decisions for feature ideas sourced from upstream
[navidrome/navidrome Discussions](https://github.com/navidrome/navidrome/discussions), evaluated specifically for
this fork. Each entry below was scoped by reading the actual discussion, researching what already exists in this
codebase that the feature could reuse, and estimating effort/value before deciding whether to build it. Where a
feature has shipped, effort estimates are checked against what it actually took to build — useful signal for
scoping the next one.

**Effort** is relative to this fork's own past work (podcasts ≈ Large, folder browsing ≈ Large, skip songs ≈ Small).
**Value** is judged by user demand visible in the source discussion (reply count, duplicate asks) plus how directly
it serves this fork's own use (Cirque compatibility, personal library workflow).

---

## At a glance

**7 shipped · 4 planned, ready to build · 3 in the backlog (assessed, not prioritized)**

Nothing below is currently mid-build — everything is either done, or scoped-but-not-started. When something is
picked up, move it into its own "🔨 In progress" section at the top so it's visible at a glance.

### ✅ Shipped (7)

| Feature | Source | Effort (est. → actual) |
|---|---|---|
| Skip / auto-pass disliked songs | [#3899](https://github.com/navidrome/navidrome/discussions/3899) | Small → Small |
| Genre exploration page + sidebar entry | [#4656](https://github.com/navidrome/navidrome/discussions/4656), [#4249](https://github.com/navidrome/navidrome/discussions/4249), [#2631](https://github.com/navidrome/navidrome/discussions/2631) | Medium → Medium |
| Genre merging (admin-defined aliases, any-player sync) | User follow-up request | Medium → Medium |
| User-defined song tagging + smart-playlist integration | [#4823](https://github.com/navidrome/navidrome/discussions/4823) | Large → Large |
| Podcast support (Subsonic API) | own project, [PODCAST_PLAN.md](PODCAST_PLAN.md) | Large → Large |
| Physical folder browsing | own project, [navidrome-folder-roadmap.md](navidrome-folder-roadmap.md) | Large → Large |
| Enhanced scrobble attribution (Pulse integration) | own project | Small → Small |

### 📋 Planned — scoped, ready to build (4)

| Feature | Source | Effort | Value |
|---|---|---|---|
| Remove/prevent duplicate playlist tracks | [#4206](https://github.com/navidrome/navidrome/discussions/4206) | Small (exact-dup) / Medium (fuzzy cross-album) | Medium–High |
| Playlist "consume mode" (auto-remove on finish) | [#3276](https://github.com/navidrome/navidrome/discussions/3276) | Small–Medium | Low–Medium |
| AI-based auto-tagging/classification (as a plugin) | [#3145](https://github.com/navidrome/navidrome/discussions/3145); plugin itself built separately, see [AI-Auto-Tagging-Plugin](https://github.com/RFLundgren/AI-Auto-Tagging-Plugin) | Small (write path, this repo) | Medium |
| Classical music: show Work/Movement in the web UI | [#2953](https://github.com/navidrome/navidrome/discussions/2953) | Medium | Medium (niche but well-served by existing data) |

Also planned, tracked in a separate doc rather than duplicated here: **Podcast Phase 4** — resume playback
position, a cross-channel "Up Next" queue, and OPML import/export. See
[PODCAST_PLAN.md](PODCAST_PLAN.md#phased-roadmap) for the full writeup.

### 💡 Backlog — assessed, not prioritized (3)

| Feature | Source | Why it's parked |
|---|---|---|
| Last.fm play count / loved status sync | [#3454](https://github.com/navidrome/navidrome/discussions/3454) | Real feedback-loop risk (see write-up) with no existing pull-path to build on; better fits as a plugin someone opts into than a core feature |
| Sidecar tag files (`tags.yml` overrides) | [#3181](https://github.com/navidrome/navidrome/discussions/3181) | Niche audience relative to effort; revisit if there's a concrete personal need |
| Bulk multi-select / batch actions (Album, Artist lists) | mentioned in [#4249](https://github.com/navidrome/navidrome/discussions/4249) | Not yet scoped in depth; distinct from the genre work it surfaced alongside |

---

## ✅ Shipped

### Skip / auto-pass disliked songs

**Source:** [#3899](https://github.com/navidrome/navidrome/discussions/3899) — mark individual songs to be automatically
passed over during playback, without deleting them or needing a separate playlist.

**Status:** Shipped. Per-user `skipped`/`skippedAt` on the existing `annotation` table, `skip`/`unskip` Subsonic
endpoints (fork-specific, same tier as `reportPlayback`), dimmed-but-still-playable row in the web UI, and
queue-aware auto-advance in the player. Client integration contract for Cirque written up in
`SKIP_CLIENT_INTEGRATION.md` (gitignored, not published).

**Effort — estimated vs. actual:** Estimated Small going in, on the reasoning that the per-user annotation storage
shape and the smart-playlist criteria engine already existed and just needed extending. That held for the backend
(migration + model + repo + Subsonic handlers took one focused pass) and the initial UI work. What the estimate
missed: the player's in-memory queue only captured a song's skip status *at the moment it was queued* — flagging a
song already sitting in the live queue had no effect on the "next" button. That required a second pass (propagate
toggles into the live Redux queue, re-check on playback advance, still honor an explicit manual click on a flagged
row). Net: still Small, but "does the auto-skip actually re-check live state, not just at queue-build time" is now
a standing question to ask early on similar features, not an afterthought.

**Pros:** Fully self-contained, no schema surprises, reuses `starred`/`rating`'s proven end-to-end pattern
exactly. Cirque gets it too via the same Subsonic-adjacent endpoints (see `SKIP_CLIENT_INTEGRATION.md`).

**Cons:** Non-spec — only this fork's own web UI and Cirque will ever honor it; any other Subsonic client just sees
an inert extra field. No smart-playlist criteria integration yet (can't build a `.nsp` rule filtering out skipped
songs) — deliberately deferred, would be a small follow-up given `starred`/`rating` are already criteria fields.

---

### Genre exploration page + sidebar entry

**Source:** [#4656](https://github.com/navidrome/navidrome/discussions/4656) ("Genres 'exploration' Page") +
[#4249](https://github.com/navidrome/navidrome/discussions/4249) ("Include GENRE menu in Left Menu") +
[#2631](https://github.com/navidrome/navidrome/discussions/2631) ("Genre list/grid based on standard React Admin
template") — three discussions asking for the same underlying feature across four years. Pick a genre → land on a
"genre homepage" with Albums, Top Songs, and a shuffle-this-genre action, reachable from a real sidebar entry
instead of the current Albums-then-filter workaround.

**Status:** Shipped, plus a "Create Playlist" action and a colored chip dashboard added beyond the original
discussion scope (see below).

- **Genre index page.** Confirmed the near-free assessment was correct, with one wrinkle: `model.Genre.SongCount`/
  `AlbumCount` were tagged `json:"-"`, so native REST never actually returned them, even though
  `persistence/sql_tags.go`'s `baseTagRepository.newSelect()` already joins `library_tag` and computes both counts
  correctly (`genreRepository.selectGenre()` just appends `name` on top via squirrel's additive `.Columns()`, it
  doesn't replace the aggregate columns). One two-line JSON-tag fix, not a new query — the "close to free"
  assessment held, just not for the reason originally assumed.
- **Per-genre page.** Albums (`genre_id` filter, already registered on the `album` resource), Top Songs
  (`genre_id` filter + the already-registered `play_count` sort key), and Recently Added all needed zero new
  backend code — confirmed, not assumed. Shuffle reuses `ShuffleAllButton` as-is.
- **Sidebar entry.** Flows through `Menu.jsx`'s existing generic resource loop automatically once `list`/`show`
  are set on the `genre` Resource — no hardcoded `MenuItemLink` needed (unlike Folders, which needs one for its
  settings-gated toggle).
- **Create Playlist** (added during this session's scoping discussion, beyond #4656/#4249/#2631's original ask):
  N random deduplicated tracks from a genre, with an "exclude skipped songs" option. Server-side, not
  fetch-everything-then-dedup-in-JS — `GetRandom` (already composes with arbitrary filters) over-fetches
  candidates, a new `core/matcher.DeduplicateMediaFiles` (new file, same package as the existing `Matcher`, reusing
  its private Jaro-Winkler/duration-proximity scoring helpers) clusters by MBID → ISRC → fuzzy title/artist/
  duration, keeping the highest-bitrate copy per cluster. Verified with an actual standalone Go program (not just
  hand-tracing — `core/matcher` builds locally, unlike `persistence`/`server`, so real execution was possible): 9
  constructed test cases covering MBID/ISRC/fuzzy clustering, correct non-clustering of covers and duration
  mismatches, order preservation, and edge cases all passed.
- **Colored gradient chip dashboard** (added later, replacing the original plain text list): each genre gets a
  deterministic hash-to-gradient chip showing name, song count, and album count at a glance.

**What was cut from scope during this session's discussion, deliberately:**
- **Artist-by-genre / Playlist-by-genre** sections — still genuinely unsupported (no backend query exists), same
  gap identified in the original assessment. Not built.
- **Image-grid genre index** (cover-art collages) — still needs genre artwork that doesn't exist. The colored chip
  dashboard shipped instead.
- **Genre listening stats** — considered and cut; belongs in the Pulse companion app, which already has richer
  scrobble-attribution data (see below) for exactly this purpose.
- **"Start Radio"** (infinite/self-extending queue) — considered and cut after review; "Shuffle This Genre" already
  gives a large (500-track) randomized queue with zero new backend work, and the infinite-queue mechanism wasn't
  judged worth the extra complexity without a demonstrated real gap.

**Effort — estimated vs. actual:** Estimated Medium going in; held up as Medium. The genuinely new pieces were, as
predicted, the aggregation page itself and (newly in scope) the dedup-clustering function — everything else really
was wiring together filters/sorts that already existed, confirmed by direct code reading rather than assumption at
every step (the `SongCount`/`AlbumCount` investigation in particular could easily have gone the other way — the
first research pass concluded the opposite of what direct reading of `persistence/sql_tags.go` showed).

---

### Genre merging (admin-defined aliases, any-player sync)

**Source:** Direct follow-up request after the Genre exploration page shipped — the genre index can get cluttered
with near-duplicate genres ("Hip-Hop" / "Hip Hop" / "HipHop") from inconsistently-tagged files. The user already
has a genre-merge feature built client-side in Cirque and wants Navidrome to become the authoritative source, so
Cirque and any other Subsonic client see one consistent merged view instead of each client maintaining its own
local merge.

**Status:** Shipped, including a later revision adding multi-select merge and an Edit view (see below).

- **Architecture decision, reversed once during scoping.** Genre matching happens in two structurally different
  places: an ID-based world (native REST, `tagIDFilter` against the `tag`/`library_tag` tables) and a name-based,
  no-ID world (Subsonic's `getGenres`/`getSongsByGenre`/`getRandomSongs` — what Cirque actually uses — plus the
  smart-playlist criteria engine, both matching directly against the scanner's `tags` JSON column). A read-time
  alias overlay (the first plan) couldn't make the Subsonic surface merge-aware without separate logic, so it was
  dropped in favor of canonicalizing genre values at the single choke point where they're already cleaned:
  `sanitize()` in `model/metadata/metadata.go`, called once per tag value per file during scanning. This flows
  through automatically to `MediaFile.Tags`, `Album.Tags` (pure re-aggregation, not a re-parse), and the
  `tag`/`library_tag` tables — zero other query-layer changes needed, and "any player in sync" falls out for free
  since every Subsonic client already reads from Navidrome's normal API.
- **`genre_alias` table** — a flat string-to-string mapping (`alias_name` → `canonical_name`), deliberately not an
  FK, since canonicalization happens before any tag ID exists. The repository (`persistence/genre_alias_repository.go`)
  flattens chains on write (merging into an existing alias resolves through to its final canonical target) and
  repoints existing rows on write (if the alias being created was itself used as another row's canonical target,
  those rows get repointed too) — so the mapping is always exactly one level deep, and cyclic merges are rejected.
- **Scan-time hook** — `model/metadata/genre_aliases.go` (new), an `atomic.Pointer`-backed alias map mirroring how
  `model.TagMappings()` already provides config-like data to the same cleaning pipeline, loaded once per scan run
  (not per file) by `scanner/scanner.go` before Phase 1 runs.
- **Admin UI, revised after initial ship:** the original single alias→canonical Create form was replaced with a
  "Merge Genres" UI supporting multi-select (merge several source genres into one target in a single action, still
  supporting a brand-new not-yet-scanned target name), plus an Edit view so an existing merge can be re-pointed or
  corrected without deleting and recreating it. No backend changes were needed for Edit — the repository already
  supported `Update()` with the same chain-flatten/repoint validation as `Save()`.

**Important caveat, initially mis-documented then corrected:** a merge only takes effect once each affected file's
tags are actually re-read. A normal quick Scan Now skips files whose mtime on disk hasn't changed — which is every
file already in the library — so **a Full Scan is required** to apply a new merge to existing data (new/changed
files pick it up automatically on their next normal scan either way).

**Verification:** the scan-time hook (`sanitize()`'s new genre-canonicalization branch) was verified with an
actual standalone Go program exercising the real `metadata.New()` pipeline end-to-end (`model/metadata` builds
locally, unlike `persistence`/`scanner`) — 7 cases covering passthrough, simple aliasing, multiple aliases to one
canonical, and cleared aliases all passed. The repository's chain-flatten/repoint/self-merge logic (in
`persistence`, which can't be built locally due to this environment's known `db` package blocker) was hand-traced
against 6 scenarios instead: simple merge, chaining onto an existing alias, repointing existing rows whose target
becomes an alias, direct self-merge rejection, and a cyclic self-merge caught via chain resolution.

---

### User-defined song tagging + smart-playlist integration

**Source:** [#4823](https://github.com/navidrome/navidrome/discussions/4823) — per-user custom tags on songs
(independent of embedded file metadata), a "Bind by Tags" bulk-add-to-playlist action, and tags usable as
smart-playlist criteria so playlists auto-update as tags change.

**Status:** Shipped, in three phases, matching the original plan almost exactly:
- **Phase 1 — core tagging.** New `media_file_tag` table (flat, per-user, no separate "tag entity" — a tag starts
  existing the first time it's applied and stops existing when unused, like a hashtag), native REST endpoints, and
  a tag-picker dialog (modeled on the existing playlist-picker's "select existing or create new inline" pattern)
  wired into the song context menu next to Add to Playlist.
- **Phase 2 — smart-playlist criteria.** A new `usertag` criteria field, evaluated via an `EXISTS` subquery scoped
  to the playlist owner — reused the exact per-owner scoping mechanism `rating`/`starred` criteria already use, so
  a `.nsp` smart playlist can now auto-update based on a user's own tags.
  Registering per-user tag *names* as individual criteria fields turned out not to work (the field registry is a
  single global map populated once at startup) — the fix was one generic `usertag` field whose *value* is the tag
  name, with per-user isolation living entirely in the SQL layer instead.
- **Phase 3 — Bind by Tags.** A "My Tag" filter on the song list plus a "Bind by Tag" button, reusing the existing
  add-to-playlist dialog rather than building a new one — fetch songs matching the filter, then open the standard
  picker with those IDs pre-selected.

**Effort — estimated vs. actual:** Estimated Large going in; held up as Large, not a surprise in either direction.
The two pieces expected to be free (smart-playlist criteria reuse, per-user storage shape reuse) mostly were, though
Phase 2 needed more surgery than planned: the criteria SQL generator's helper functions (`isNotExpr`, `missingExpr`,
`likeExpr`, `comparisonExpr`, `rangeExpr`) were plain functions with no access to the playlist owner, not methods —
threading the owner ID through required touching six call sites, not the "just add a new cond type" the plan
assumed. Caught and fixed cleanly since it was verified by hand-tracing every test case against the generated SQL
before pushing (local Go builds are blocked in this environment by a pre-existing, unrelated sqlite3-driver issue),
and CI confirmed the trace was correct on the first attempt. Two real bugs did surface in CI, both from routine
interface-surface gaps rather than the tagging logic itself: adding `MediaFileTag` to the `DataStore` interface
broke `tests/mock_data_store.go` (needed the same stub method every other repository accessor has), and an existing
test asserting "every non-tag/non-role field has a `smartPlaylistFields` entry" needed to also exclude the new
`IsUserTag` fields.

**Pros:** Confirms the roadmap's original read was right — the per-user annotation-table shape and the
already-tag-aware smart-playlist criteria engine really were the two hardest pieces, and reusing them instead of
inventing new ones kept this tractable despite being the largest single feature since podcasts/folder browsing.

**Cons:** As anticipated, the tag-editor UI and write-path were genuinely new surface with no existing pattern to
copy directly — mitigated by modeling the picker closely on the existing playlist-picker dialog instead of
designing from scratch.

**Follow-up unlocked:** the AI auto-tagging plugin idea ([#3145](https://github.com/navidrome/navidrome/discussions/3145))
is still blocked — see that entry — but its prerequisite (a fork-owned, non-scanner tag table) now exists. Making
`media_file_tag` plugin-writable would need a new host-service capability, not built as part of this feature.

---

### Podcast support (Subsonic API)

**Status:** Shipped (Phases 1–3). Full design writeup, including what's still on the roadmap (Phase 4: resume
playback position, a cross-channel "up next" queue, OPML import/export), see [PODCAST_PLAN.md](PODCAST_PLAN.md).

### Physical folder browsing

**Status:** Shipped. For the full history of what's shipped and what's planned, see
[navidrome-folder-roadmap.md](navidrome-folder-roadmap.md).

### Enhanced scrobble attribution (Pulse integration)

**Status:** Shipped. `client`/`source`/`origin`/`playback_mode` fields on every scrobble/play report, exposed to
plugins via the same `ScrobbleRequest`/`NowPlayingRequest` types, for this fork's own Pulse companion project. See
the [README](README.md#enhanced-scrobble-attribution-pulse-integration) for details.

---

## 📋 Planned — scoped, ready to build

### Remove/prevent duplicate playlist tracks

**Source:** [#4206](https://github.com/navidrome/navidrome/discussions/4206) — detect and remove duplicate songs
from a playlist (e.g. the same recording appears on both a studio album and a "Best Of" compilation, ending up
added twice as two different files), plus optionally prevent duplicates from being added in the first place. A
supporting reply cited Ampache having this already and called out the practical pain of managing it by hand in
playlists with thousands of songs. (Cirque already does client-side dedup at playlist-creation time; this is
purely a server-side/web-UI gap.)

**Status:** Scoped, not started.

**Effort: Small for exact duplicates, Medium for the "smart" cross-album case.** Confirmed the gap is real and
total — playlists have **zero** dedup protection today: adding the same exact MediaFile ID twice just creates two
rows, no unique constraint beyond row position. The good news for the harder version of this feature: the
"same song, different file" identity problem is already solved elsewhere in the codebase, just for a different
purpose. `core/matcher` implements exactly this — MBID exact match → ISRC exact match → fuzzy title/artist
fallback (Jaro-Winkler similarity + duration proximity + a 6-tier scoring system, configurable threshold) — as a
generic core service, not plugin-coupled. It currently matches *external* song descriptions against local tracks
(used for agent recommendations), so its heuristics would need repurposing into a "cluster these playlist tracks by
likely-same-recording" utility rather than being callable verbatim — real adaptation work, but the hard
algorithmic thinking (how do you decide two files are the same song) doesn't need reinventing. Both the UI slot
(`PlaylistActions.jsx`'s existing toolbar, alongside Shuffle/Export/etc.) and the removal mechanism
(`RemoveTracks`, an existing efficient bulk-delete-by-position primitive) are trivial — no new mechanism needed
for either.

**Pros:** The exact-duplicate tier is close to free (no algorithm needed, just a membership check before insert/on
demand) and immediately useful. The harder cross-album tier has real prior art to build from rather than a blank
page, unlike most "detect similarity" features would.

**Cons:** MBID/ISRC-based matching only catches duplicates in well-tagged libraries (Picard-tagged files reliably
have MBID; ISRC is spottier) — libraries without consistent tagging would fall through entirely to the fuzzy-title
matcher, meaning the "clean" exact-identifier matches are likely the minority case for a lot of real libraries.

**Recommendation:** Ship exact-duplicate detection/prevention first (small, immediately useful, catches the common
"added the same album twice" case) as its own pass; treat cross-album same-recording detection (adapting
`core/matcher`'s heuristics) as a follow-up rather than bundling both into one release.

---

### Playlist "consume mode" (auto-remove on finish)

**Source:** [#3276](https://github.com/navidrome/navidrome/discussions/3276) — a playlist mode where each track is
automatically removed once it finishes playing, so a curated queue (e.g. "these two albums") drains as you listen
instead of staying static. The maintainer suggested a Smart Playlist workaround (`playCount:0` filter); the OP
correctly pushed back that this filters the *whole library*, not just the tracks they curated, and that re-adding
a track should reset its consumed status — something a query-based Smart Playlist can't do since it has no real
add/remove semantics.

**Status:** Scoped, not started.

**There's a better existing-feature composition than what was suggested in the discussion.** The smart-playlist
criteria engine already has `inPlaylist`/`notInPlaylist` operators (confirmed in `model/criteria/operators.go`) —
combining `{"inPlaylist": "your-static-playlist-id"}` with `{"is": {"playCount": 0}}` gives a genuinely scoped,
auto-shrinking "consuming" view using **zero new code**, unlike the maintainer's suggested filter alone. The catch:
smart playlists are cached with a refresh delay (`SmartPlaylistRefreshDelay`), so a track wouldn't visibly vanish
the instant it finishes, only on next evaluation — not real-time, but it does solve the actual stated problem
(avoid replaying already-heard tracks from a curated set).

**Effort: Small–Medium for the "full" literal-removal version**, if the smart-playlist composition above isn't
close enough. Confirmed `reportPlayback`/`scrobble` carry no playlist context today, so this isn't a server-side
scrobble hook — it'd be client-side player logic, structurally identical to how skip-songs got built: a new
`consume` boolean on `Playlist`, a UI toggle, and a reactive hook in `Player.jsx` on track-finish that calls the
*existing* remove-track endpoint (`RemoveTracks`, already used for the duplicate-cleanup feature above) when the
current playlist is flagged consume-mode. One real wrinkle: playlist-track removal is position-based, not
stable-ID-based, and removing a track renumbers everything after it — auto-removing during sequential playback
needs to account for that drift (re-fetch position state after each removal, not trust a client-cached position
number from queue-build time).

**Pros:** The "full" version resolves the OP's second complaint (re-adding doesn't reset consumed status) for
free — a real playlist already supports arbitrary add/remove, no special reset logic needed once it's genuine
playlist behavior instead of a filtered view. Reuses the exact same removal primitive and player-hook pattern as
two features already shipped/scoped above.

**Cons:** Lower demand signal than genre, skip-songs, or duplicate-cleanup — single discussion, modest reply
count. The position-drift handling during auto-removal is a genuine (if contained) wrinkle to get right.

**Recommendation:** Worth mentioning the `inPlaylist`+`playCount` smart-playlist composition in the discussion
itself regardless of whether the "full" version ever gets built — it's a real, already-available answer nobody
in the thread suggested. Build the full version only if the near-real-time gap of the smart-playlist composition
turns out to matter in practice.

---

### AI-based auto-tagging/classification (as a plugin)

**Source:** [#3145](https://github.com/navidrome/navidrome/discussions/3145) — auto-classify tracks by genre,
language, mood, etc. using an AI service, so the whole library becomes filterable by AI-suggested tags instead of
manually maintained playlists per genre/language.

**Status:** Reassessed 2026-07-18, after `#4823` shipped — no longer blocked. Split into two pieces with the split
resolved 2026-07-20: the plugin itself lives in its own repo,
[AI-Auto-Tagging-Plugin](https://github.com/RFLundgren/AI-Auto-Tagging-Plugin) (not yet public), not in
`plugins/examples/` here — it only talks to Navidrome through the plugin API, the same reasoning that already put
Cirque and Pulse in their own repos rather than this one. Full design doc, build plan, and the open product
decision (should AI tags be private-per-user or shared library-wide) live there, not duplicated here.

**What's left in this repo:** a small prerequisite, tracked here since it's core-fork work — three new
Subsonic-tier endpoints (`setUserTag`, `removeUserTag`, `getUserTags`) so a plugin can write to the existing
`media_file_tag` table, mirroring `skip`/`unskip` exactly (`server/subsonic/media_annotation.go`, registered in
`server/subsonic/api.go`'s existing authenticated route group). Zero new persistence code — those repository
methods already scope via `loggedUser(ctx)` internally. **Effort: Small. Not started.**

---

### Classical music: show Work/Movement in the web UI

**Source:** [#2953](https://github.com/navidrome/navidrome/discussions/2953) — "It would be nice if we could get a
working system for works, movements and movement names just like Qobuz's webplayer." Low direct engagement (1
comment), but that comment points at [navidrome/navidrome#2601](https://github.com/navidrome/navidrome/pull/2601)
("Classical music metadata support"), a substantial upstream PR that's been open since November 2023 and is still
active (last updated within the last month) — real, ongoing community interest in this, just not concentrated in
the discussion thread itself.

**Status:** Scoped, not started. More is already in place than the open discussion/PR suggests.

**What's already here, inherited from upstream:** confirmed by reading the current codebase directly, not
assumption — `work`, `movementname`, `movement`, and `movementtotal` are already-registered tag mappings
(`resources/mappings.yaml`), `model.MediaFile.Movements()` already builds structured `Movement{Name, Number, Count}`
data from those tags (`model/mediafile.go`), and it's already serialized on the Subsonic/OpenSubsonic API response
(`server/subsonic/helpers.go` → `responses.Child.Movements`, with test coverage confirming spec-compliant JSON
output). So any OpenSubsonic client that renders `movements` already gets classical structure today, on the version
this fork is currently synced to.

**What's actually missing:** confirmed there is **no** `classical`/`IsClassical` flag anywhere in the backend (`grep`
across `model`, `persistence`, `server/subsonic` for "classical" turns up nothing but an unrelated comment) — the
auto-detection, composer-as-additional-artist, and `classical` smart-playlist-criteria pieces described in PR
#2601 have not landed. Nor has PR #2601's web UI piece: "Displays the Work Name + Composers/Arrangers in the album
tracklist (Navidrome Web UI only, not Subsonic API)" — confirmed via a repo-wide grep for "movement" that the web
UI (`ui/src/`) never references it. So Navidrome's own web player is the one place a classical listener doesn't see
any of this, despite the data already flowing through the API for everyone else.

**Effort: Medium**, and deliberately scoped narrower than PR #2601's full ask to avoid the same trap that's
apparently kept that PR from merging for two and a half years:
- **In scope:** surface `work`/movement data in the album tracklist UI (`ui/src/album/AlbumSongs.jsx` and
  `SongTitleField`) — show Work name as a grouping/subheading when present, and prefer "Movement N. Movement Name"
  over the plain Title field for tracks that have movement tags. No new backend Go code needed for the data itself,
  since parsing already exists — only exposing `work`/movement fields on the native REST song response the web UI
  actually reads from (currently these live in the generic raw/mapped tags blob, not as first-class JSON fields on
  `Child`/song the way `genre` is).
- **Deliberately out of scope for v1:** auto-detecting "is this album classical" (PR #2601's `is_classical`/
  `showmovement`-tag-or-heuristic logic), promoting composers to additional artists, and a `classical` smart-playlist
  criteria field. All three are real, useful ideas, but each is its own design decision with room for disagreement
  (the PR's own author flagged the genre-name-based heuristic as something they "don't 100% like") — bundling them
  into a first pass risks the same two-and-a-half-year stall. Gate v1 purely on "does this track have work/movement
  tags at all," no classification heuristic required.

**Pros:** Real, load-bearing tag data already exists and is already tested/serialized correctly on the API side —
this is a presentation gap, not a from-scratch metadata project. Zero risk to non-classical libraries (tracks
without work/movement tags render exactly as they do today).

**Cons:** Niche audience — classical listeners are a minority of self-hosters, and the discussion itself has low
direct engagement. The "right" UI treatment (grouping by work vs. just relabeling the track title) has real design
judgment calls that PR #2601's screenshots only partially resolve for this fork's own component structure.

**Recommendation:** Worth building as a contained v1 (tracklist display only, no auto-detection or smart-playlist
criteria) — the data's already there and tested, so this is mostly UI work. Revisit the auto-detect/composer/smart-
playlist pieces later, independently, if v1 gets real use.

---

## 💡 Backlog — assessed, not prioritized

### Last.fm play count / loved status sync

**Source:** [#3454](https://github.com/navidrome/navidrome/discussions/3454) — pull `userplaycount`/`userloved`
from Last.fm's API back into Navidrome, so listening history survives a library replacement/re-rip. A community
member already built this as an *offline* Python script (NaviSync) that requires the server to be stopped.

**Status:** Assessed as a **plugin**, not a core feature — and specifically NOT buildable as a plugin without a
real design decision about the play-count backfill mechanism (see Cons).

**Effort: Medium, as a plugin.** The plugin system (WASM/Extism, sandboxed, in-process) already provides
everything needed at the infrastructure level: `http` permission for calling Last.fm's API, `scheduler` permission
for a daily cron-style sync job (a documented example pattern already), and `subsonicapi` + `users` permissions for
writing back into Navidrome via its own internal Subsonic API (no direct DB access is ever exposed to plugins, by
design).

**Pros:** Genuinely good fit for the plugin system — better than a core PR, since it's inherently
per-user/optional/external-service-dependent, exactly what plugins exist for. `star`/`unstar` maps cleanly onto
Last.fm's `userloved` — a clean, absolute set, no design problem there.

**Cons:** There is no "set play_count to N" anywhere in this codebase (plugin or core) — only `IncPlayCount`,
which always adds exactly +1. Backfilling a count of, say, 340 means firing 340 synthetic `scrobble` calls with
fabricated historical timestamps. Worse: a scrobble event fans out to *every* registered scrobbler, not just the
plugin issuing it — a naive backfill plugin would re-scrobble all those synthetic plays straight back to the
user's real Last.fm account if they also have outbound scrobbling enabled, inflating the exact number it's trying
to fix. No existing code to build on for the actual Last.fm data-pull either — the built-in Last.fm integration
(`adapters/lastfm/`) is outbound-only (push scrobbles, fetch metadata/similarity); nothing calls `track.getInfo`.

**Recommendation:** Not planned for this fork currently — flagging the feedback-loop risk here in case anyone
attempts it, since it's the kind of bug that wouldn't show up until a user with active outbound scrobbling runs a
backfill and quietly corrupts their own Last.fm history.

---

### Sidecar tag files (`tags.yml` overrides)

**Source:** [#3181](https://github.com/navidrome/navidrome/discussions/3181) — store metadata overrides in a
separate file next to the media file instead of editing embedded tags, so corrections don't touch the original
source files. A community fork (`tagfiles-bfr`) already built a more elaborate version with glob patterns and CEL
expression transforms.

**Status:** Assessed, not planned.

**Effort: Medium**, scoped to the simple version (flat per-track/per-folder key-value overrides, no glob/CEL). The
codebase answers the two scariest architectural questions favorably: the tag-merge pipeline (`RawTags` → `clean()`
→ `ToMediaFile()`) is source-agnostic, so a sidecar's key-values could merge in before `metadata.New()` runs with
no restructuring; and there's already a directly-reusable precedent — this fork's own lyrics sidecar support
already parses a YAML sidecar format (`.lrc`/`.srt`/`.ttml` lookup, same-folder/same-basename convention). The one
real gap: track-level rescan currently keys only on the *audio file's own* mtime, so a sidecar-only edit wouldn't
trigger re-import on a quick scan without a small, contained fix to that check.

**Pros:** Structurally smaller than it looks — two of the three hardest questions (merge pipeline assumptions,
folder-walk visibility into non-audio files) already resolve favorably, with real sidecar precedent to copy rather
than invent.

**Cons:** Niche audience relative to effort — the community's own engagement level on this discussion is lower
than genre or skip-songs, and the full-featured version people actually seem to want (glob patterns, CEL
expression transforms) is a much bigger, more speculative scope than the tractable v1 described here.

**Recommendation:** Not prioritized — revisit if there's a concrete personal need for it (e.g. correcting tags on
files you don't want to touch directly), since the core mechanism is genuinely low-risk to add later.

---

### Bulk multi-select / batch actions across list views

**Source:** mentioned in passing in [#4249](https://github.com/navidrome/navidrome/discussions/4249) — multi-select
across Album, Artist, and Song pages with batch playback/action options.

**Status:** Noted, not scoped. Song lists already have bulk-select (`SongBulkActions`/`SongDatagrid`); Album/Artist
list views likely don't have the equivalent. Not investigated in depth — flagged here only because it surfaced
alongside the genre discussions and is a distinct feature, not genre-specific. Deliberately **not** folded into
the Genre exploration page above — no real coupling between the two, and combining them would turn a moderate
feature into a sprawling one for no benefit.

**Recommendation:** Scope separately if/when there's interest — would need its own research pass into how far
Album/Artist list views are from Song's existing bulk-select pattern.
