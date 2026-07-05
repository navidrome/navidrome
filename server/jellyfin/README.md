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
`http://localhost:4533/jellyfin/System/Info/Public`).

## Authentication

Jellyfin clients authenticate with `POST /Users/AuthenticateByName` using the user's Navidrome
username/password, and get back an `AccessToken` (a Navidrome JWT). That token is then sent on
every subsequent request as the `X-Emby-Token` header (or embedded in the
`X-Emby-Authorization`/`Authorization` header's `Token="..."` field, or as an `api_key` query
param — all three forms are accepted, matching what different clients do).

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
| Browsing | `GET Items`, `GET Users/{userId}/Items`, `GET Items/{itemId}`, `GET Users/{userId}/Items/{itemId}`, `GET Users/{userId}/Items/Latest` |
| Artists / genres | `GET Artists`, `GET Artists/AlbumArtists`, `GET Genres`, `GET MusicGenres` |
| Images | `GET Items/{itemId}/Images/{type}[/{index}]` (public, not library-scoped) |
| Favorites / ratings | `POST`/`DELETE Users/{userId}/FavoriteItems/{itemId}`, `POST`/`DELETE Users/{userId}/Items/{itemId}/Rating` |
| Streaming | `GET Audio/{itemId}/stream[.{container}]`, `GET Audio/{itemId}/universal`, `GET`/`POST Items/{itemId}/PlaybackInfo` |
| Playback reporting | `POST Sessions/Playing`, `POST Sessions/Playing/Progress`, `POST Sessions/Playing/Stopped`, `POST Sessions/Capabilities[/Full]` |
| Playlists | `POST Playlists`, `GET Playlists/{playlistId}/Items`, `POST`/`DELETE Playlists/{playlistId}/Items` |

Any other path returns a `404` with a `{}` JSON body, and is logged server-side at `Debug` level
as `Jellyfin API: unhandled route` (method + path). If a client you're testing needs an endpoint
that isn't in the table above, check the server logs for these lines to see exactly what it's
requesting.

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

# 10. Create a playlist, add the song, then remove it
PLAYLIST_ID=$(curl -s -X POST "${AUTH[@]}" -H 'Content-Type: application/json' \
  -d "{\"Name\":\"My Playlist\",\"Ids\":[\"$SONG_ID\"]}" "$BASE/Playlists" | jq -r .Id)
curl -s -X POST "${AUTH[@]}" "$BASE/Playlists/$PLAYLIST_ID/Items?Ids=$SONG_ID"
ENTRY_ID=$(curl -s "${AUTH[@]}" "$BASE/Playlists/$PLAYLIST_ID/Items" | jq -r '.Items[0].PlaylistItemId')
curl -s -X DELETE "${AUTH[@]}" "$BASE/Playlists/$PLAYLIST_ID/Items?EntryIds=$ENTRY_ID"
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
- **`PlaylistItemId` round-trip is untested against a live client.** `GET
  Playlists/{id}/Items` tags each entry with `PlaylistItemId` (the playlist-track row id, not
  the song id) specifically so a client can echo it back via
  `DELETE Playlists/{id}/Items?EntryIds=...` to remove one occurrence of a song that appears
  more than once in the same playlist. This is exercised by unit tests, but hasn't been
  confirmed against a real Jellyfin client's actual request shape — worth a manual smoke test
  before relying on it.
- **ID handling is pass-through only.** Navidrome ids are used verbatim as Jellyfin item ids;
  there's no hex/GUID-reversible id scheme. If a client turns out to mangle ids in a way that
  requires one (e.g. expecting a GUID shape), that will need a follow-up change.
