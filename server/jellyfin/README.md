# Jellyfin API

This package implements a subset of the [Jellyfin](https://jellyfin.org/) REST API on top of
Navidrome's existing library, users, playlists and scrobbling infrastructure. It lets
Jellyfin-compatible clients (e.g. [Finamp](https://github.com/jmshrv/finamp),
[jftui](https://github.com/Aanok/jftui)) browse and stream a Navidrome library without
requiring a real Jellyfin server.

It is **not** a full Jellyfin server implementation: only the endpoints needed to browse a music
library, stream audio, manage favorites/ratings for songs, albums, artists, and playlists, report
playback, and manage playlists are implemented. Video, live TV, plugins, and Jellyfin's
admin/dashboard APIs are out of scope.

## Enabling

The Jellyfin API is disabled by default. Enable it via `navidrome.toml`:

```toml
[Jellyfin]
Enabled = true
# Optional: override the server name reported to clients (defaults to "Navidrome <version>")
ServerName = "My Music Server"
# Optional: usernames to show in the client login user-picker (default: none). See "Public user list".
ExposedPublicUsers = "alice, bob"
# Optional: answer LAN auto-discovery broadcasts on UDP 7359 (default: false). See "Auto discovery".
AutoDiscovery = true
# Optional: let users sign in new devices with a 6-digit code (default: true). See "Quick Connect".
QuickConnect = false
# Optional: max collection responses streaming at once (default: half the DB connection pool,
# min 2). Each streaming response holds a DB connection for its whole duration; excess requests
# queue rather than fail.
MaxConcurrentStreams = 4
```

or via environment variables:

```bash
ND_JELLYFIN_ENABLED=true
ND_JELLYFIN_SERVERNAME="My Music Server"
ND_JELLYFIN_EXPOSEDPUBLICUSERS="alice,bob"
ND_JELLYFIN_AUTODISCOVERY=true
ND_JELLYFIN_QUICKCONNECT=false
ND_JELLYFIN_MAXCONCURRENTSTREAMS=4
```

Once enabled, the API is mounted at:

```
http://<host>:<port>/jellyfin
```

All the paths below are relative to that base URL (e.g. `System/Info/Public` means
`http://localhost:4533/jellyfin/System/Info/Public`). Routes are matched **case-insensitively**,
since real Jellyfin clients (and `jellyfin-apiclient-python`) send mixed-case paths.

## Auto discovery

With `AutoDiscovery = true`, Navidrome answers the Jellyfin LAN discovery broadcast
(`who is JellyfinServer?` on UDP port 7359), so clients list the server without a typed URL.
It is off by default because a real Jellyfin server on the same host owns that port. If the port
is taken, Navidrome logs a warning and keeps running without discovery.

The advertised address is `BaseURL` when it includes a host. Otherwise it is the bind `Address`
when that is a specific IP, or else the local IP that faces the requesting client, plus `Port`. With a
unix socket `Address` there is no port to advertise, so discovery only starts when `BaseURL` has a host.

Discovery answers on all IPv4 interfaces, but the advertised address follows `BaseURL`, `Address` and
`Port`. If `Address` is a loopback or a single interface IP and `BaseURL` has no host, clients on other
networks get an address they cannot reach. Set `BaseURL` to the address clients should use.

Docker: use host networking (`network_mode: host` / `--network host`). On Linux, bridge mode does not
deliver broadcasts to the container, even with `-p 7359:7359/udp`, so clients never find the server.
With host networking the server sees the host's IP, so `BaseURL` is not needed for discovery. If host
networking is not an option, leave discovery off. Keep UDP 7359 on the LAN: never forward it from the internet.

## Authentication

Jellyfin clients authenticate with `POST /Users/AuthenticateByName` using the user's Navidrome
username/password, and get back an `AccessToken` (a Navidrome JWT). That token is then sent on
every subsequent request as the `X-Emby-Token` header (or embedded in the
`X-Emby-Authorization`/`Authorization` header's `Token="..."` field, or as an `api_key`/`ApiKey`
query param — all forms are accepted, matching what different clients do).

`POST /Users/AuthenticateByName` is rate-limited per IP with the same limiter as the native
`/auth/login` (`AuthRequestLimit`/`AuthWindowLength`), since it's an unauthenticated brute-force
surface.

Access tokens do not expire, matching real Jellyfin. They are revoked by a password change, which bumps the user's token epoch.

### Quick Connect

Quick Connect signs a new device in without typing a password. The client shows a 6-digit code,
a signed-in user approves it, and the client gets its `AccessToken`. It is on by default
(`Jellyfin.QuickConnect`); when off, `GET QuickConnect/Enabled` returns `false` and every other
Quick Connect call returns 401.

1. `POST QuickConnect/Initiate` (public; needs `Client`, `Device`, `DeviceId` and `Version` in the
   auth header) returns the `Code` and a `Secret`.
2. The user approves the code, either in the Navidrome web UI (user menu → **Quick Connect**, which
   shows the app and device before approving) or from a signed-in Jellyfin client with
   `POST QuickConnect/Authorize?Code=`. Admins may pass `UserId` to approve for another user.
3. The client polls `GET QuickConnect/Connect?Secret=` until `Authenticated` is `true`.
4. `POST Users/AuthenticateWithQuickConnect` with `{"Secret": "..."}` returns the same result as
   `AuthenticateByName`.

Pending codes live in memory and expire after 10 minutes (a server restart drops them). Unlike
Jellyfin, a secret signs in only once. `Initiate`, `AuthenticateWithQuickConnect` and code approval
(`Authorize` and the web UI) are rate-limited per IP like the login; `Connect` is not, since some
clients poll it every second.

### Public user list (login picker)

`GET /Users/Public` lets a client render a login user-picker (tap a user, then just type the
password) instead of a blank username field. It's **unauthenticated**, so by default it exposes
**no** users. Set `Jellyfin.ExposedPublicUsers` to a comma-separated list of usernames to advertise:

```toml
[Jellyfin]
ExposedPublicUsers = "alice, bob"
```

Only the named users are listed (never the full user table), resolved live per request; a configured
name that doesn't exist is skipped and logged at `Warn`. Each entry is a minimal DTO (`Name`, `Id`)
with no `Policy`/`Configuration`, so admin status isn't leaked to unauthenticated callers, and no
avatar (`PrimaryImageTag` omitted — Navidrome has no per-user profile images).

## Players and sessions

Every authenticated request registers (or refreshes) the calling device as a Navidrome player,
mirroring Subsonic's `getPlayer` — so a Jellyfin client shows up in the players list (and scrobbling
has a player) as soon as it makes any authenticated call, not only when it reports playback. The
player id is the device id from `X-Emby-Authorization` (`DeviceId="..."`); the player name is
`Client [Device]`. Those field values are URL-decoded, since some clients percent-encode them
(Jellify sends `Device="Pixel%208%20Pro"`, Finamp sends it raw). A request that carries no
client/device info (e.g. the `GET socket` handshake, which authenticates via `?api_key=` only) is
skipped, so it doesn't create a nameless player.

## Multi-library behavior

Jellyfin has no native concept of multiple music libraries the way Navidrome does, so each
Navidrome library the current user can access is exposed as its own top-level Jellyfin
"CollectionFolder" view (`GET /UserViews`), instead of merging every library into a single view.
Browsing (`/Items`), artists, and the "Latest" list are all scoped to the libraries the
authenticated user has access to. Fetching an item the user cannot access returns `404`, never
`403`, so ids can't be used as an existence oracle. An inaccessible library id sent as `ParentId`
is not a `404`: it is simply not treated as a library, so none of that library's content is
returned.

### Browsing filters

`GET /Items` accepts the filter params clients use to build screens: `ParentId` (a library view id
for scoping, an artist id when browsing into an artist's albums, or an album id when browsing into
an album's tracks); `AlbumArtistIds`/`ArtistIds`/`contributingArtistIds` (an artist's albums or
tracks — Finamp's artist screen sends these *alongside* `ParentId=<libraryId>`); `AlbumIds` (an
album's tracks — Feishin fetches them this way instead of `ParentId`); `GenreIds` (a
genre's albums or tracks — Finamp's genre screen sends it the same way; `/Artists/AlbumArtists`
and `MusicArtist` queries accept it too, matching artists credited on an album of that genre);
`Years`; `StudioIds` (record labels, as listed by `GET Studios`); `SearchTerm`;
`Filters` (`IsFavorite`, `IsFavoriteOrLikes`, `IsPlayed`, `IsUnplayed`) and the standalone
`isFavorite`/`isPlayed` booleans it can also be expressed as — `Filters` wins when both are sent, as
in Jellyfin; `Likes`, `Dislikes`, `IsFolder`, `IsNotFolder` and `IsResumable` have no Navidrome
equivalent and are ignored; `SortBy`/`SortOrder` (every recognized key is applied in order, so secondary keys break ties;
unrecognized keys are skipped, and `Random` always sorts alone);
`StartIndex`/`Limit`; and `Ids` (batch fetch by id). `Recursive=false` with a library `ParentId`
returns direct children only (no tracks — no track is a library's direct child).

## Implemented endpoints

| Area | Endpoints |
|---|---|
| Handshake / system | `GET System/Info/Public`, `GET System/Info` (authenticated), `GET`/`POST System/Ping`, `GET System/Endpoint` (authenticated) |
| Quick Connect | `GET QuickConnect/Enabled`, `POST QuickConnect/Initiate`, `GET QuickConnect/Connect`, `POST QuickConnect/Authorize` (authenticated), `POST Users/AuthenticateWithQuickConnect` |
| Auth | `POST Users/AuthenticateByName`, `GET Users/Public` |
| Users | `GET UserViews`, `GET Users/{userId}/Views`, `GET Users/Me`, `GET Users/{userId}` |
| Browsing | `GET Items`, `GET Users/{userId}/Items`, `GET Items/{itemId}`, `GET Users/{userId}/Items/{itemId}`, `GET Items/Latest`, `GET Users/{userId}/Items/Latest`, `DELETE Items/{itemId}` (playlists only) |
| Artists / genres / labels | `GET Artists`, `GET Artists/AlbumArtists`, `GET Genres`, `GET MusicGenres`, `GET Studios`, `GET Items/Filters` |
| Similar / mixes | `GET Artists/{itemId}/Similar`, `GET Items/{itemId}/Similar`, `GET Albums/{itemId}/Similar`, `GET {Items,Songs,Albums,Artists,Playlists}/{itemId}/InstantMix`, `GET Artists/InstantMix?id=`, `GET MusicGenres/InstantMix?id=` |
| Images | `GET Items/{itemId}/Images/{type}[/{index}]` (public), `POST`/`DELETE Items/{itemId}/Images/{type}` (playlist cover, authenticated) |
| Favorites / ratings for songs, albums, artists, and playlists | `POST`/`DELETE UserFavoriteItems/{itemId}`, `POST`/`DELETE Users/{userId}/FavoriteItems/{itemId}`, `POST`/`DELETE Users/{userId}/Items/{itemId}/Rating`, `GET UserItems/{itemId}/UserData`, `GET Users/{userId}/Items/{itemId}/UserData` |
| Streaming | `GET Audio/{itemId}/stream[.{container}]`, `GET Audio/{itemId}/universal`, `GET Audio/{itemId}/main.m3u8`, `GET Items/{itemId}/File`, `GET Items/{itemId}/Download`, `GET`/`POST Items/{itemId}/PlaybackInfo` (`HEAD` too on stream, universal, File, Download and images; a transcode HEAD answers without starting it) |
| Lyrics | `GET Audio/{itemId}/Lyrics` |
| Playback reporting | `POST Sessions/Playing`, `POST Sessions/Playing/Progress`, `POST Sessions/Playing/Stopped`, `POST Sessions/Playing/Ping` (no-op), `POST Sessions/Capabilities[/Full]` |
| Playlists | `POST Playlists`, `GET Playlists/{playlistId}`, `POST Playlists/{playlistId}` (rename / visibility / replace tracks), `GET Playlists/{playlistId}/Items`, `POST`/`DELETE Playlists/{playlistId}/Items` (`POST` honors `position`), `POST Playlists/{playlistId}/Items/{entryId}/Move/{newIndex}`, `GET Playlists/{playlistId}/Users[/{userId}]` |
| Real-time | `GET socket` (WebSocket; keeps clients like Finamp from 404-loop-reconnecting) |
| AudioMuse-AI (see below) | `GET AudioMuseAI/info`, `GET AudioMuseAI/health`, `GET AudioMuseAI/similar_tracks`, `GET AudioMuseAI/find_path` |

Any other path returns a `404` with a `{}` JSON body, and is logged server-side at `Debug` level
as `Jellyfin API: unhandled route` (method + path). If a client you're testing needs an endpoint
that isn't in the table above, check the server logs for these lines to see exactly what it's
requesting.

## Playlist management

Playlists are the main writable surface of this API:

- **Container expansion.** When creating (`POST Playlists`), adding to (`POST Playlists/{id}/Items`)
  or replacing (`POST Playlists/{id}`) a playlist, the `Ids` may contain **containers** — album,
  artist or playlist ids — not just song ids. Each is expanded into its tracks (in order) before
  the write, matching how Jellyfin clients populate these lists. A bare song id passes through.
- **Id list encoding.** `POST`/`DELETE Playlists/{id}/Items` accept the id list both ways clients
  spell it: repeated params (`ids=X&ids=Y`, how Jellify's `@jellyfin/sdk` serializes arrays) and a
  single comma-separated value (`ids=X,Y`, Finamp). Reading only the first value would add just one
  track of an expanded album.
- **Update** (`POST Playlists/{id}`): with `Ids` present, the track list is **replaced** (Finamp
  uses this for reordering) — an explicit empty `Ids` (`[]`) **clears** the playlist, while an
  omitted `Ids` leaves the tracks untouched and only updates `Name`/`IsPublic`. `IsPublic` maps to
  Navidrome's `Public` flag, surfaced to clients as `OpenAccess` on `GET Playlists/{id}`.
- **Cover art**: `POST Items/{id}/Images/Primary` uploads a playlist cover (raw or base64 body,
  JPEG/PNG/WebP/GIF detected by magic number, extension from `Content-Type`); `DELETE` removes it.
  Only playlists are writable through this API — album/artist covers come from tag/sidecar scanning,
  so a non-playlist id returns `501`. Uploads honor the same gates as the native endpoint: they're
  bounded by `MaxImageUploadSize` and require `EnableArtworkUpload` for non-admins.
- **`PlaylistItemId`**: `GET Playlists/{id}/Items` tags each entry with `PlaylistItemId` (the
  playlist-track row id, distinct from the song id) so a client can echo it back via
  `DELETE Playlists/{id}/Items?EntryIds=...` to remove one occurrence of a song that appears more
  than once in the same playlist.

Ownership is enforced by `core/playlists`: a non-owner editing/deleting a playlist gets `403` if
it is visible to them (public) or `404` if it is not (private) — the API never reveals that
someone else's private playlist exists.

## Images

The `GET Items/{itemId}/Images/{type}` route is intentionally **public** (artwork isn't sensitive,
matching Jellyfin's lenient image handling), so it carries no authenticated user. Artwork is
therefore resolved under an **elevated admin context** — the same approach `core/artwork`'s cache
warmer uses — so user-scoped items like private playlists still resolve their cover instead of
falling back to the placeholder. Album, artist, media-file and playlist ids are all resolved to
their Navidrome `ArtworkID`.

## Item ids are GUIDs

Jellyfin item ids are GUIDs, serialized as 32 lowercase hex chars with no dashes
(`Guid.ToString("N")`). Navidrome ids are canonical 22-char base62 encodings of a 128-bit value,
so `dto.EncodeID`/`dto.DecodeID` map between the two via `model/id` — losslessly except for the
~2⁻⁹⁶ chance an id's 128-bit value falls in the reserved space below (leading 12 bytes all zero).

Three emitted ids aren't 128-bit values: integer library ids, the synthetic playlists folder, and
`PlaylistItemId` (a playlist *entry position* — `playlist_tracks.id` is an `integer` column).
They use a reserved GUID space — 12 zero bytes, a non-zero kind tag, a 24-bit payload — so
library `1` is `00000000000000000000000001000001`. The tag is never zero, because Jellyfin
serializes the all-zero GUID as `null`.

`DecodeID` accepts dashed and uppercase GUIDs (Jellyfin's `Guid.Parse` does) and returns
`ok=false` for anything malformed — including "" — which handlers surface as a 404.

The wire format must stay GUID-shaped for this reason: Finamp's saved-queue persistence bit-packs
each item id into exactly 16 bytes (`packIds()` in `lib/models/finamp_models.dart`), so a 32-hex
GUID round-trips exactly, whereas a longer id would be silently truncated.

## Streaming and transcoding

The stream endpoints reuse the same transcode-decision pipeline as the Subsonic `/stream` endpoint:

- **`GET Audio/{id}/stream[.{container}]` / `universal`** — the target format comes from the
  `.{container}` path suffix, the `container` param, or (when neither is present) `audioCodec`.
  `audioBitRate`/`maxStreamingBitrate` are bits/sec, per Jellyfin convention. `static=true`
  forces direct play (raw), never a transcode.
- **`GET Items/{id}/File` / `Download`** — always the original file bytes, matching real Jellyfin.
  Finamp plays through `File` when its transcoding setting is off, so an undecodable format (e.g.
  DSF) can't be rescued server-side on this path.
- **`GET Audio/{id}/main.m3u8`** — the endpoint Finamp plays through when its transcoding setting
  is on. Implemented as a single-segment HLS VOD playlist whose one segment is the progressive
  transcode endpoint above, so the whole pipeline (decision, cache, forced transcoding) is reused.
  Segment codec honors `audioCodec` but is limited to what HLS packed-audio can carry (`aac`,
  `mp3`); anything else falls back to `aac`. Seeking re-reads from the start, like Subsonic
  transcoded streams.
- **Server-forced transcoding.** A format/bitrate configured on the registered player (Settings →
  Players) is applied to `stream`, `universal` and `main.m3u8` — same override semantics as
  Subsonic. `File`/`Download` stay raw. For HLS clients, force `aac` or `mp3`; other formats are
  advertised and served but packed-audio players won't decode them.

## Lyrics

`GET Audio/{id}/Lyrics` serves the main lyric track as a `LyricDto` (`Start` in 100ns ticks,
word-level `Cues` when present), resolved through the full `core/lyrics` pipeline (embedded, `.lrc`
sidecars, plugins per `LyricsPriority`). No lyrics returns `404`, never an empty `200`.

Results, misses included, are cached for 5 minutes: Jellify fetches lyrics for every played track and
Feishin on every song change, so lyric-less tracks are the hot path. Concurrent misses on the same
track share one pipeline run. That run is detached from the request, so a cancelled request doesn't
fail it for other waiters, and its context has a one-minute deadline.

Finamp opens its lyrics view only when the track has a `Lyric` `MediaStream` (`HasLyrics` is just a
list badge). Browse lists set both from embedded lyrics only. `PlaybackInfo` runs the full pipeline
per track, so sidecar and plugin lyrics show up there. Feishin also requires server version ≥ 10.9
(we advertise 12.1.0).

## AudioMuse-AI compatible endpoints

Compatibility shim for Jellyfin front-ends that integrate [AudioMuse-AI](https://github.com/NeptuneHub/audiomuse-ai-plugin)
— e.g. [Symfonium](https://symfonium.app/) can use these endpoints for sonic mixes when
connected as a Jellyfin client.
Backed natively by Navidrome's `core/sonic` engine (the `SonicSimilarity` plugin capability) — no
external AudioMuse-AI backend or proxy is involved. The endpoints are gated on a `SonicSimilarity`
plugin being loaded, like the Subsonic `sonicSimilarity` OpenSubsonic extension.

- `GET /AudioMuseAI/info` — returns `{"Version": <navidrome version>, "AvailableEndpoints": [...]}` (200).
  `AvailableEndpoints` lists the endpoints below only when a provider is loaded; otherwise it is empty.
- `GET /AudioMuseAI/health` — liveness probe: 200 with an empty body when a provider is loaded, else 404.
- `GET /AudioMuseAI/similar_tracks?item_id=<id>&n=10&eliminate_duplicates=true` — 404 when no provider is
  loaded; otherwise a JSON array of `{author, distance, item_id, title}` (200; `[]` when there is no match
  or no `item_id`). `eliminate_duplicates` (default true) limits results to one track per artist.
- `GET /AudioMuseAI/find_path?start_song_id=<id>&end_song_id=<id>&max_steps=25` — 404 when no provider is
  loaded; otherwise `{"path": [{author, item_id, title, tempo?}], "total_distance": <float>}` (200), or 400
  with `start_song_id and end_song_id are required.` when either id is missing.

`item_id`/`start_song_id`/`end_song_id` are the GUID-form ids Navidrome hands Jellyfin clients.
`tempo` comes from the track's BPM when known; the richer AudioMuse per-track features
(`energy`, `key`, `mood_vector`, `scale`, `other_features`) are not provided. In multi-library
setups, `find_path`'s `path` and `total_distance` only reflect hops through tracks in libraries
the caller can access, since hops through inaccessible libraries are filtered out of the result.

## curl walkthrough

This mirrors the sequence a real client (e.g. Finamp) follows: handshake, login, browse the
library hierarchy, fetch playback info, stream, favorite, report playback, and manage a playlist.

```bash
BASE=http://localhost:4533/jellyfin

# 1. Handshake (no auth required)
curl -s "$BASE/System/Info/Public" | jq .

# 2. Login - capture the AccessToken
TOKEN=$(curl -s -X POST "$BASE/Users/AuthenticateByName" \
  -H 'Content-Type: application/json' \
  -d '{"Username":"admin","Pw":"password"}' | jq -r .AccessToken)

AUTH=(-H "X-Emby-Token: $TOKEN")

# 3. List the user's views (one per accessible library)
curl -s "${AUTH[@]}" "$BASE/UserViews" | jq .

# 4. Browse artists
curl -s "${AUTH[@]}" "$BASE/Items?IncludeItemTypes=MusicArtist" | jq .
ARTIST_ID=$(curl -s "${AUTH[@]}" "$BASE/Items?IncludeItemTypes=MusicArtist&Limit=1" | jq -r '.Items[0].Id')

# 5. Drill into that artist's albums (ParentId with no IncludeItemTypes defaults to MusicAlbum)
ALBUM_ID=$(curl -s "${AUTH[@]}" "$BASE/Items?ParentId=$ARTIST_ID" | jq -r '.Items[0].Id')

# 6. List the album's songs
USER_ID=$(curl -s "${AUTH[@]}" "$BASE/Users/Me" | jq -r .Id)
SONG_ID=$(curl -s "${AUTH[@]}" "$BASE/Users/$USER_ID/Items?ParentId=$ALBUM_ID&IncludeItemTypes=Audio" \
  | jq -r '.Items[0].Id')

# 7. Ask for playback info, then stream the song
curl -s -X POST "${AUTH[@]}" "$BASE/Items/$SONG_ID/PlaybackInfo" | jq .
curl -s "${AUTH[@]}" "$BASE/Audio/$SONG_ID/stream" -o /tmp/song.audio

# 8. Favorite the song
curl -s -X POST "${AUTH[@]}" "$BASE/Users/$USER_ID/FavoriteItems/$SONG_ID" | jq .

# 9. Report playback start/stop (also drives scrobbling)
curl -s -X POST "${AUTH[@]}" -H 'Content-Type: application/json' \
  -d "{\"ItemId\":\"$SONG_ID\",\"PositionTicks\":0}" "$BASE/Sessions/Playing"
curl -s -X POST "${AUTH[@]}" -H 'Content-Type: application/json' \
  -d "{\"ItemId\":\"$SONG_ID\",\"PositionTicks\":1200000000}" "$BASE/Sessions/Playing/Stopped"

# 10. Create a playlist from a whole album (the album id is expanded to its tracks)
PLAYLIST_ID=$(curl -s -X POST "${AUTH[@]}" -H 'Content-Type: application/json' \
  -d "{\"Name\":\"My Playlist\",\"Ids\":[\"$ALBUM_ID\"]}" "$BASE/Playlists" | jq -r .Id)

# 11. Make it public, then remove one entry
curl -s -X POST "${AUTH[@]}" -H 'Content-Type: application/json' \
  -d '{"IsPublic":true}' "$BASE/Playlists/$PLAYLIST_ID"
ENTRY_ID=$(curl -s "${AUTH[@]}" "$BASE/Playlists/$PLAYLIST_ID/Items" | jq -r '.Items[0].PlaylistItemId')
curl -s -X DELETE "${AUTH[@]}" "$BASE/Playlists/$PLAYLIST_ID/Items?EntryIds=$ENTRY_ID"

# 12. Delete the playlist
curl -s -X DELETE "${AUTH[@]}" "$BASE/Items/$PLAYLIST_ID"
```

## Testing

Handler-level unit tests live alongside each file (`*_test.go`). A full end-to-end suite in
[`e2e/`](e2e) exercises every endpoint through the real router against a real SQLite database and
real repositories (only artwork, streaming, ffmpeg, external metadata agents and sonic similarity
are stubbed), with per-`Describe` snapshot isolation — mirroring the Subsonic `server/subsonic/e2e`
suite. Run it with:

```bash
make test PKG=./server/jellyfin/...
```

## Known limitations

- **Genre lists ignore `ParentId`.** `GET Genres`/`MusicGenres` and
  `Items?IncludeItemTypes=MusicGenre` list the genres of every library the user can access, never
  only the `ParentId` library. `Items?IncludeItemTypes=MusicGenre` also ignores `SearchTerm`.
  `GET Items/Filters` is the exception: its genres follow `ParentId`.
- **Search skips some filters.** With `SearchTerm`, `Filters=IsFavorite`, `IsPlayed`, `IsUnplayed`
  and `isFavorite`/`isPlayed` are skipped, so the response holds every search match. The full-text
  search's first phase has no annotation join to filter on. Artist searches also skip the role (album
  artist vs. artist) and `GenreIds`. Playlist lists ignore `SearchTerm`.
- **Search pages are capped at 2,000 items.** A larger `Limit` is lowered to 2,000. Jellyfin has no
  cap.
- **One-character searches return nothing.** The shared search layer ignores terms shorter than two
  characters, so album, artist and song searches for `a` come back empty. Real Jellyfin has no
  minimum.
- **Playlist entry ids are positions, not song ids.** `PlaylistItemId` encodes the entry's position
  in the playlist, so ids change when entries are inserted or removed; re-read the list after an
  edit. Real Jellyfin uses the song id. Clients that echo `PlaylistItemId` back (Finamp) work;
  clients that send the song id as `EntryIds` or to `Move` (Jellify) get `404`.
- **Rating uses the `rating` param.** `POST Users/{userId}/Items/{itemId}/Rating` reads a `rating`
  (0-10) and stores it as 0-5 stars. Jellyfin's `likes` boolean is not read, so a client that sends
  only `likes` clears the rating.
- **Missing endpoints.** `UserPlayedItems` (mark played/unplayed), `Search/Hints` and the non-legacy
  `UserItems/{itemId}/Rating` are not implemented. Play counts only change through playback
  reporting.
- **Some `Fields` are never emitted.** `ProviderIds`, `People`, `Etag` and `DateLastMediaAdded` are
  accepted but never returned.
- **Blurhashes can be missing.** `ImageBlurHashes` carries the real blurhash that the artwork
  pipeline computes and stores. Until that happens for an item, the field is omitted: a fake value
  would pin the wrong placeholder in clients that key their image cache on it.
- **The WebSocket only keeps alive; it pushes no events.** `GET socket` sends `ForceKeepAlive` and
  answers `KeepAlive` pings, so real-time clients (Finamp) keep a working session instead of
  reconnecting in a loop. It never pushes session, playstate or library-change events.
- **List lyric badges only see embedded lyrics.** Tracks with only sidecar or plugin lyrics show no
  `HasLyrics` badge in lists, because those sources can't be known at list time without per-row I/O.
  `PlaybackInfo` and `Audio/{id}/Lyrics` still find them (see "Lyrics").
