# Jellyfin API

This package implements a subset of the [Jellyfin](https://jellyfin.org/) REST API on top of
Navidrome's existing library, users, playlists and scrobbling infrastructure. It lets
Jellyfin-compatible clients (e.g. [Finamp](https://github.com/jmshrv/finamp),
[jftui](https://github.com/dylanmtaylor/jftui)) browse and stream a Navidrome library without
requiring a real Jellyfin server.

It is **not** a full Jellyfin server implementation: only the endpoints needed to browse a music
library, stream audio, manage favorites/ratings, report playback, and manage playlists are
implemented. Video, live TV, plugins, and Jellyfin's admin/dashboard APIs are out of scope.

## Enabling

The Jellyfin API is disabled by default. Enable it via `navidrome.toml`:

```toml
[Jellyfin]
Enabled = true
# Optional: override the server name reported to clients (defaults to "Navidrome <version>")
ServerName = "My Music Server"
```

or via environment variables:

```bash
ND_JELLYFIN_ENABLED=true
ND_JELLYFIN_SERVERNAME="My Music Server"
```

Once enabled, the API is mounted at:

```
http://<host>:<port>/jellyfin
```

All the paths below are relative to that base URL (e.g. `System/Info/Public` means
`http://localhost:4533/jellyfin/System/Info/Public`). Routes are matched **case-insensitively**,
since real Jellyfin clients (and `jellyfin-apiclient-python`) send mixed-case paths.

## Authentication

Jellyfin clients authenticate with `POST /Users/AuthenticateByName` using the user's Navidrome
username/password, and get back an `AccessToken` (a Navidrome JWT). That token is then sent on
every subsequent request as the `X-Emby-Token` header (or embedded in the
`X-Emby-Authorization`/`Authorization` header's `Token="..."` field, or as an `api_key`/`ApiKey`
query param — all forms are accepted, matching what different clients do).

## ID encoding

Navidrome item ids are **hex-encoded at the API boundary** (`dto.EncodeID`/`DecodeID`): every id
is hex-encoded on the way out and hex-decoded on the way in. This is required because some clients
parse ids as radix-16 — Finamp's queue `packIds`, for instance, does `int.parse(chunk, radix:16)`,
which chokes on Navidrome's base-62 nanoids (e.g. `5QFKvMsJrd57QE2Le2dKKo`). Because a raw MD5 id
from an old migrated library is itself valid hex, correctness depends on every emit path encoding
and every receive path decoding — see `dto/ids.go`.

## Multi-library behavior

Jellyfin has no native concept of multiple music libraries the way Navidrome does, so each
Navidrome library the current user can access is exposed as its own top-level Jellyfin
"CollectionFolder" view (`GET /UserViews`), instead of merging every library into a single view.
Browsing (`/Items`), artists, and the "Latest" list are all scoped to the libraries the
authenticated user has access to; a library (or item within it) the user cannot access returns
`404`, never `403`, so ids can't be used as an existence oracle.

## Implemented endpoints

| Area | Endpoints |
|---|---|
| Handshake / system | `GET System/Info/Public`, `GET`/`POST System/Ping`, `GET QuickConnect/Enabled` |
| Auth | `POST Users/AuthenticateByName`, `GET Users/Public` |
| Users | `GET UserViews`, `GET Users/{userId}/Views`, `GET Users/Me`, `GET Users/{userId}` |
| Browsing | `GET Items`, `GET Users/{userId}/Items`, `GET Items/{itemId}`, `GET Users/{userId}/Items/{itemId}`, `GET Users/{userId}/Items/Latest`, `DELETE Items/{itemId}` (playlists only) |
| Artists / genres | `GET Artists`, `GET Artists/AlbumArtists`, `GET Genres`, `GET MusicGenres` |
| Images | `GET Items/{itemId}/Images/{type}[/{index}]` (public), `POST`/`DELETE Items/{itemId}/Images/{type}` (playlist cover, authenticated) |
| Favorites / ratings | `POST`/`DELETE Users/{userId}/FavoriteItems/{itemId}`, `POST`/`DELETE Users/{userId}/Items/{itemId}/Rating` |
| Streaming | `GET Audio/{itemId}/stream[.{container}]`, `GET Audio/{itemId}/universal`, `GET Items/{itemId}/File`, `GET Items/{itemId}/Download`, `GET`/`POST Items/{itemId}/PlaybackInfo` |
| Playback reporting | `POST Sessions/Playing`, `POST Sessions/Playing/Progress`, `POST Sessions/Playing/Stopped`, `POST Sessions/Capabilities[/Full]` |
| Playlists | `POST Playlists`, `GET Playlists/{playlistId}`, `POST Playlists/{playlistId}` (rename / visibility / replace tracks), `GET Playlists/{playlistId}/Items`, `POST`/`DELETE Playlists/{playlistId}/Items`, `GET Playlists/{playlistId}/Users[/{userId}]` |
| Real-time | `GET socket` (WebSocket; keeps clients like Finamp from 404-loop-reconnecting) |

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
- **Update** (`POST Playlists/{id}`): with `Ids` present, the track list is **replaced** (Finamp
  uses this for reordering); otherwise `Name` and/or `IsPublic` are updated. `IsPublic` maps to
  Navidrome's `Public` flag, surfaced to clients as `OpenAccess` on `GET Playlists/{id}`.
- **Cover art**: `POST Items/{id}/Images/Primary` uploads a playlist cover (raw or base64 body,
  extension from `Content-Type`); `DELETE` removes it. Only playlists are writable through this
  API — album/artist covers come from tag/sidecar scanning, so a non-playlist id returns `501`.
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
real repositories (only artwork/streaming/ffmpeg are stubbed), with per-`Describe` snapshot
isolation — mirroring the Subsonic `server/e2e` suite. Run it with:

```bash
make test PKG=./server/jellyfin/...
```

## Known limitations

- **Genres are global.** `GET Genres`/`MusicGenres` is not scoped to the current user's
  libraries (genre tags aren't per-library entities in Navidrome's model).
- **Artist item-access relies on list-time scoping.** Unlike albums and songs (which each
  belong to exactly one library and are checked against `user.HasLibraryAccess` on every
  fetch), an artist can have content across multiple libraries via `library_artist`, so there's
  no single library id to gate a direct `GET Items/{artistId}` or favorite/rating call against.
  Access control for artists is enforced by scoping the `Artists`/`Items?IncludeItemTypes=MusicArtist`
  *list* to the user's libraries, plus the persistence layer's own defense-in-depth; a client
  that already has an artist id from elsewhere is not re-checked against library membership.
- **Album track order.** Browsing an album's tracks (`Items?ParentId=<albumId>&IncludeItemTypes=Audio`)
  without an explicit `SortBy` does not guarantee track order — clients that need it should pass a
  sort. Track numbers are always present on each item (`IndexNumber`).
- **Playlists never match `Filters=IsFavorite`.** `GET Items?IncludeItemTypes=Playlist` (alone
  or mixed with other types, e.g. Finamp's favorites screen sending
  `IncludeItemTypes=Audio,MusicAlbum,Playlist`) is supported, but `model.Playlist` has no
  starred/annotation concept, so a favorites query always returns zero playlists rather than
  erroring.
- **MD5-hash ids from old migrated libraries.** The hex id codec assumes ids are opaque; a raw
  32-char MD5 id is itself valid hex and so must be encoded/decoded symmetrically like any other.
  This is handled, but is the most fragile id case — see the note in `dto/ids.go`.
