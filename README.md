<a href="https://www.navidrome.org"><img src="resources/logo-192x192.png" alt="Navidrome logo" title="navidrome" align="right" height="60px" /></a>

# Navidrome Music Server &nbsp;[![Tweet](https://img.shields.io/twitter/url/http/shields.io.svg?style=social)](https://twitter.com/intent/tweet?text=Tired%20of%20paying%20for%20music%20subscriptions%2C%20and%20not%20finding%20what%20you%20really%20like%3F%20Roll%20your%20own%20streaming%20service%21&url=https://navidrome.org&via=navidrome)

[![Build](https://img.shields.io/github/actions/workflow/status/RFLundgren/navidrome_experimental/pipeline.yml?branch=master&logo=github&style=flat-square)](https://github.com/RFLundgren/navidrome_experimental/actions)
[![Docker Image](https://img.shields.io/badge/ghcr.io-navidrome__experimental-blue?logo=docker&style=flat-square)](https://github.com/RFLundgren/navidrome_experimental/pkgs/container/navidrome_experimental)
[![Dev Chat](https://img.shields.io/discord/671335427726114836?logo=discord&label=discord&style=flat-square)](https://discord.gg/xh7j7yF)
[![Subreddit](https://img.shields.io/reddit/subreddit-subscribers/navidrome?logo=reddit&label=/r/navidrome&style=flat-square)](https://www.reddit.com/r/navidrome/)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0-ff69b4.svg?style=flat-square)](CODE_OF_CONDUCT.md)

Navidrome is an open source web-based music collection server and streamer. It gives you freedom to listen to your
music collection from any browser or mobile device. It's like your personal Spotify!

## About This Fork

Navidrome is already the best self-hosted alternative to Spotify for your music library. **navidrome-experimental**
takes that same server and gives it a second life as a podcast platform too — full RSS subscriptions, streaming,
downloads, and retention, through the exact same Subsonic API you already use for your songs. No separate podcast
app, no separate sync, no separate account. It also adds physical folder browsing, for anyone who's spent years
organizing music by hand and doesn't want that structure hidden behind a metadata-only view.

Everything else works exactly like upstream Navidrome — same installation, same configuration, same Subsonic
compatibility, same plugin system. This fork just adds:

- 🎙️ **[Podcast support](PODCAST_PLAN.md)** — full RSS subscriptions, streamed or downloaded, through the real
  Subsonic API. See [below](#podcast-support-experimental) for the full feature list.
- 📁 **[Physical folder browsing](navidrome-folder-roadmap.md)** — navigate, play, and manage your library exactly
  as it's laid out on disk. See [below](#physical-folder-browsing-experimental) for the full feature list.
- 🏷️ **User-defined song tagging** — private per-user labels on songs, independent of file metadata, with
  tag-based filtering, bulk playlist add, and smart-playlist criteria support. See
  [below](#user-defined-song-tagging-experimental) for details.
- ⏭️ **Skip / auto-pass disliked songs** — flag a song as skipped and the player automatically passes over it during
  playback, without deleting it. See [below](#skip--auto-pass-disliked-songs-experimental) for details.
- 📡 **Enhanced scrobble attribution** — richer client/source/playback-mode context on every scrobble, available to
  plugins too. See [below](#enhanced-scrobble-attribution-pulse-integration) for details.
- 🎼 **Genre exploration** — a real sidebar entry for browsing by genre, with albums, top songs, and one-click
  deduplicated playlist creation. See [below](#genre-exploration-experimental) for details.
- 🔗 **Genre merging** — collapse near-duplicate genres from inconsistent tagging into one, applied at scan time so
  every Subsonic client and smart playlist sees the merge too, not just this web UI. See
  [below](#genre-merging-experimental) for details.

Kept in sync with upstream: currently based on [Navidrome v0.63.2](https://github.com/navidrome/navidrome/releases/tag/v0.63.2),
merged in directly rather than maintained as a standalone patch set. Syncs happen periodically, not on a fixed
schedule — check the [releases page](https://github.com/RFLundgren/navidrome_experimental/releases) for this fork's
own tagged checkpoints (e.g. `v0.63.2-experimental.3`), which pin the exact upstream baseline plus the fork-specific
fixes each one includes.

### Getting navidrome-experimental

This isn't in the official Navidrome image — you'll need to pull this fork's image specifically. Two tags are
published:

- `:stable` — the newest [tagged release](https://github.com/RFLundgren/navidrome_experimental/releases). Updated
  only when a new checkpoint is cut, not on every commit. **Recommended for most people.**
- `:develop` — tracks the tip of `master` directly. Gets fixes sooner, but may occasionally be in flux mid-fix.

Docker Compose, using the recommended `:stable` tag:

```yaml
services:
  navidrome:
    image: ghcr.io/rflundgren/navidrome_experimental:stable
    container_name: navidrome
    ports:
      - "4533:4533"
    restart: unless-stopped
    environment:
      ND_SCANSCHEDULE: 1h
      ND_LOGLEVEL: info
      ND_SESSIONTIMEOUT: 24h
    volumes:
      - "./data:/data"
      - "/path/to/your/music:/music:ro"
```

Already running stock Navidrome? Point your existing `docker-compose.yml` at
`ghcr.io/rflundgren/navidrome_experimental:stable` instead of the official image and keep your existing `/data`
volume — this fork tracks upstream closely and only *adds* tables/migrations, so your library and settings carry
over untouched; `docker compose pull && docker compose up -d` is all it takes.

For everything else — configuration options, reverse proxy setup, environment variables, building from source — the
[Documentation](#documentation) section below and [project's website](https://www.navidrome.org/docs/) apply exactly
as they do for upstream Navidrome.

__Any feedback is welcome!__ Found a bug or have a feature idea specific to this fork's podcast/folder support?
File it on [this fork's issue tracker](https://github.com/RFLundgren/navidrome_experimental/issues) — please don't
report fork-specific issues upstream. For anything about Navidrome itself, the upstream project welcomes
[GitHub issues](https://github.com/navidrome/navidrome/issues) or discussion in their
[Subreddit](https://www.reddit.com/r/navidrome/). If you want to contribute to the upstream project in any other way 
([ui/backend dev](https://www.navidrome.org/docs/developers/), 
[translations](https://www.navidrome.org/docs/developers/translations/), 
[themes](https://www.navidrome.org/docs/developers/creating-themes)), please join the chat in their 
[Discord server](https://discord.gg/xh7j7yF). 

## Installation

For this fork specifically, see [Getting navidrome-experimental](#getting-navidrome-experimental) above. For
general installation concepts (reverse proxies, environment variables, building from source, etc.) that apply the
same way here as upstream, see instructions on the [project's website](https://www.navidrome.org/docs/installation/).

## Features
 
 - Handles very **large music collections**
 - Streams virtually **any audio format** available
 - Reads and uses all your beautifully curated **metadata**
 - Great support for **compilations** (Various Artists albums) and **box sets** (multi-disc albums)
 - **Multi-user**, each user has their own play counts, playlists, favourites, etc...
 - Very **low resource usage**
 - **Multi-platform**, runs on macOS, Linux and Windows. **Docker** images are also provided
 - Ready to use binaries for all major platforms, including **Raspberry Pi**
 - Automatically **monitors your library** for changes, importing new files and reloading new metadata 
 - Supports **lyrics** from sidecar .ttml, .yaml/.yml Lyricsfile, .elrc, .lrc, .srt, .txt files and embedded TTML, Enhanced LRC, LRC, SRT, and plain-text tags (via `lyricspriority`)
 - **Themeable**, modern and responsive **Web interface** based on [Material UI](https://material-ui.com)
 - **Compatible** with all Subsonic/Madsonic/Airsonic [clients](https://www.navidrome.org/docs/overview/#apps)
 - **Transcoding** on the fly. Can be set per user/player. **Opus encoding is supported**
 - Translated to **various languages**

## Podcast Support (Experimental)

Most self-hosted music servers treat podcasts as an afterthought, if they support them at all — usually meaning a
separate app, a separate sync, or no real download management. This fork builds podcasts as a first-class feature
on the same server, through the real Subsonic API spec's podcast endpoints — not a Navidrome-only extension. Any
client that has implemented that part of the spec gets full support with no server-specific hacks needed. Whether
your particular client shows a podcasts tab at all comes down to that client's own developers — spec coverage
varies a lot across the Subsonic app ecosystem, so check what your client actually supports before assuming.

<p align="left">
    <img width="800" src=".github/screenshots/ss-podcast-episodes.png" alt="Podcast channel with episode list, showing download status and listened tracking">
</p>

### 🔍 Discover shows without hunting for RSS URLs
Search by name, or browse live, region-specific top charts to see what's actually trending where you are — pasting
in a feed URL directly still works too, if you already know exactly what you want.

### ▶️ Stream instantly, or keep it forever — your call, per show
Every subscription gets its own download policy: **stream-only** (nothing touches your disk — episodes proxy
through the server on demand, so any client can play them without ever knowing the source URL), **auto-download
new episodes** as they publish, or **backfill and download the entire back catalog**.

<p align="left">
    <img width="800" src=".github/screenshots/ss-podcast-subscriptions.png" alt="Podcast subscriptions list, showing status and download policy per channel">
</p>

### 💾 Never worry about disk space
Set retention per channel by episode count, age, or total storage, and let oldest-downloaded-first cleanup run
automatically on the same schedule as feed refreshes. Add an episode to a playlist and it's automatically protected
from cleanup — retention will never quietly delete something you're actively queued up to listen to.

### 🎵 Episodes are real library citizens, not a bolted-on side feature
Downloaded episodes slot into regular playlists right alongside your music — reorder them, mix songs and episodes
in the same playlist, export it like any other. A checkmark shows which episodes you've already listened to,
tracked independently per user on multi-user servers — click it to mark (or unmark) an episode as listened
yourself, for whenever you downloaded it and listened somewhere else entirely.

### 🔌 Real Subsonic API coverage, not a partial implementation
`getPodcasts`, `getNewestPodcasts`, `createPodcastChannel`, `refreshPodcasts`, `deletePodcastChannel`/
`deletePodcastEpisode`, `downloadPodcastEpisode`, `markPodcastEpisodeListened`/`markPodcastEpisodeUnlistened` are
all real, spec-compliant endpoints — a client still needs its own UI to call them (subscribing, browsing episodes,
etc. are new surface area, not something existing song-browsing screens do for free). Where it *does* piggyback on
what's already there: once a client has an episode's ID, streaming and downloading it go through the exact same
`stream.view`/`download.view` endpoints it already uses for
songs — no separate playback path to build.

### 🎛️ Fine-grained control
Personal toggle to hide the Podcasts section from your own sidebar if you don't use it (same mechanism as the
Folder view toggle below) — and every setting above is per-channel, so a daily news show and a sprawling back
catalog can be managed completely differently on the same server.

<p align="left">
    <img width="800" src=".github/screenshots/ss-personal-settings.png" alt="Personal settings, showing the Show Folder View and Show Podcasts toggles">
</p>

Full design writeup, including what's still on the roadmap (resume playback position, a cross-channel "up next"
queue, OPML import/export), see [PODCAST_PLAN.md](PODCAST_PLAN.md).

## Physical Folder Browsing (Experimental)

If you've spent years curating a folder structure by hand — by label, by era, by mood, by whatever system makes
sense to you — metadata-only browsing throws all of that away. This fork adds a complete second way to navigate
your library: exactly as it sits on disk, breadcrumbs and all, with every action a metadata-based view gives you
plus a few it doesn't.

<p align="left">
    <img width="800" src=".github/screenshots/ss-folder-browse.png" alt="Folder browser at an album, showing the breadcrumb trail, folder-wide action toolbar, and per-song context menu">
</p>

### 🗂️ Browse it exactly how you built it
Hierarchical navigation with working breadcrumbs at every depth, tested past 500+ items per level. Folders get the
same visual treatment as albums — thumbnails (automatically composited from the first four albums found inside, so
even a folder full of subfolders looks right), a Grid/List view toggle, and empty sections hidden automatically
rather than cluttering the view.

### ⚡ Act on a whole folder tree at once
Play All, Shuffle, or Add to Playlist for an entire folder hierarchy — subfolders included — in a single click. No
more selecting every track by hand when you just want to queue up an entire artist's directory or a whole era of
your collection.

### 📊 Know what's actually in a folder before you open it
Every folder shows its subfolder count, song count, total physical disk size, and total play time right in the
list — at a glance, without drilling in.

### 📦 Take it with you
Download an entire physical folder as a ZIP archive directly from the toolbar, generated on-the-fly from your
existing library — perfect for backups or handing a chunk of your collection to someone else.

### 🔎 Search that stays where you are
A scoped search bar inside any folder view filters to just that folder and its children — find a specific track or
subfolder without losing your place in a large hierarchy.

### 📌 Folders that stay in sync, automatically
"Pin" any physical folder as a Navidrome playlist, and it updates itself as files are added to or removed from that
folder on disk during the next library scan. Set it up once and it stays accurate forever — no manual re-adding.

### 🧭 Jump straight to where a file lives
A "Show in Folder" action on any song or album jumps you directly to its exact physical location — useful for
tracking down duplicates, checking tag consistency, or just satisfying curiosity about where something actually
lives.

### 🔌 Works beyond the web UI too
Compatible with Subsonic clients that support physical folder browsing, so this isn't a web-only feature.

For the full history of what's shipped and what's planned, see
[navidrome-folder-roadmap.md](navidrome-folder-roadmap.md).

## User-Defined Song Tagging (Experimental)

Genre, mood, and grouping tags come from your files' embedded metadata — useful, but fixed, and shared across
everyone on the server. This fork adds a second kind of tag: ones you create yourself, entirely separate from file
metadata, and private to your own account even when other people share the same library.

<p align="left">
    <img width="800" src=".github/screenshots/ss-tags-context-menu.png" alt="Song context menu showing the new Tags option, alongside Add to Playlist, Share, and other actions">
</p>

### 🏷️ Tag anything, however you want
Apply your own free-form labels to any song from its context menu — "workout," "background music for writing,"
whatever makes sense to you. A tag doesn't need to be created ahead of time; typing a new name and applying it is
enough, and it becomes a reusable option for every other song from that point on.

<p align="left">
    <img width="500" src=".github/screenshots/ss-tags-dialog.png" alt="Tags dialog, opened from a song's context menu">
</p>

### 🔒 Yours alone, even on a shared server
Tags are scoped entirely to your own account. Two people tagging songs on the same shared library never see each
other's tags, and there's no admin-managed or global tag list to work around.

### 🎯 Filter and bulk-add in one action
A "My Tag" filter on the song list narrows to everything carrying a given tag, and the "Bind by Tag" button adds
every matching song to a playlist in one click — no selecting songs one at a time.

### 🔁 Smart playlists that follow your tags automatically
Tags are usable as smart-playlist (`.nsp`) criteria, so a playlist can auto-update as you tag or untag songs,
instead of needing to be rebuilt by hand every time something changes.

Requested in [navidrome/navidrome discussion #4823](https://github.com/navidrome/navidrome/discussions/4823).

## Skip / Auto-Pass Disliked Songs (Experimental)

Some songs in your library you never want to hear again, but don't want to delete or maintain a separate exclusion
playlist for. This fork lets you flag a song as skipped, and the player automatically passes over it whenever it
comes up next — during shuffle, a playlist, an album, anywhere.

### ⏭️ Flag it once, skip it everywhere
Mark a song as skipped from its context menu. Nothing is deleted or hidden — the song stays exactly where it is in
your library, the player just automatically advances past it during playback.

### 🔁 Takes effect immediately, even mid-session
Flagging a song already sitting in your current queue skips it right away, not just for songs added afterward.

### 👀 Still visible, just dimmed
Skipped songs stay in the song list (dimmed, not hidden) and remain fully playable with an explicit click — the
auto-skip only kicks in during normal advance/auto-play.

Requested in [navidrome/navidrome discussion #3899](https://github.com/navidrome/navidrome/discussions/3899).

## Enhanced Scrobble Attribution (Pulse Integration)

Beyond just recording that a song was played, this fork adds richer context about *how* and *where* it was played
to the scrobbling pipeline.

### 📡 Client, source, and playback-mode context
Every scrobble/play report can now carry `client`, `source`, `origin`, and `playback_mode` fields (e.g.
distinguishing "Android Auto" from "Web" from "Windows Desktop"), stored alongside the play itself.

### 🔌 Available to plugins too
The Plugin API's `ScrobbleRequest`/`NowPlayingRequest` types carry the same attribution fields, so a companion
plugin (built for this fork's own Pulse project) can build listening stats like "you mostly listen via your
Favorites mix" without needing a separate external bridge process.

## Genre Exploration (Experimental)

Genre browsing in upstream Navidrome means filtering the Albums view by genre by hand. This fork adds a real
sidebar entry: a sortable genre index, and a dedicated page per genre.

### 🎼 A genre is a real page, not a filter you have to remember
Click a genre and land on its own page — the albums in that genre, its top songs by play count, recently added
tracks, and a shuffle action, all scoped to that genre automatically.

### 🔀 Shuffle or create a playlist, right from the genre page
Shuffle queues a large randomized set of the genre's songs. "Create Playlist" goes further — pick how many tracks
you want, and it builds a real playlist for you, deduplicated so a song that appears on both the studio album and
a "Best Of" compilation only shows up once (matching by MusicBrainz Recording ID, then ISRC, then title/artist/
duration similarity for files with neither), with an option to skip anything you've already flagged as skipped.

Requested across [navidrome/navidrome discussion #2631](https://github.com/navidrome/navidrome/discussions/2631),
[#4249](https://github.com/navidrome/navidrome/discussions/4249), and
[#4656](https://github.com/navidrome/navidrome/discussions/4656).

## Genre Merging (Experimental)

Inconsistently-tagged files often produce near-duplicate genres — "Hip-Hop", "Hip Hop", and "HipHop" all showing up
as separate entries. This fork lets an admin define a merge, and applies it where genre data is actually cleaned
during scanning, so the fix isn't limited to this web UI.

### 🎯 One merge, every surface in sync
Because canonicalization happens at scan time (not as a read-time filter), the merge is visible everywhere genre
data is read from: the genre index and per-genre pages in this UI, every Subsonic-compatible client (including
Cirque), and smart-playlist criteria matching on genre.

### ⚙️ Admin-only, under Settings
Go to Settings > Genre Merges and add a mapping from the genre you want retired to the genre it should count as.
Merges take effect on each affected file's next scan — trigger a Scan Now to apply immediately. Chained merges
flatten automatically (merging B into C after A was already merged into B repoints A straight at C), and merges
that would create a cycle are rejected.

## Translations

Navidrome uses [POEditor](https://poeditor.com/) for translations, and we are always looking 
for [more contributors](https://www.navidrome.org/docs/developers/translations/)

<a href="https://poeditor.com/"> 
<img height="32" src="https://github.com/user-attachments/assets/c19b1d2b-01e1-4682-a007-12356c42147c">
</a>

## Documentation
All documentation can be found in the project's website: https://www.navidrome.org/docs. 
Here are some useful direct links:

- [Overview](https://www.navidrome.org/docs/overview/)
- [Installation](https://www.navidrome.org/docs/installation/)
  - [Docker](https://www.navidrome.org/docs/installation/docker/)
  - [Binaries](https://www.navidrome.org/docs/installation/pre-built-binaries/)
  - [Build from source](https://www.navidrome.org/docs/installation/build-from-source/)
- [Development](https://www.navidrome.org/docs/developers/)
- [Subsonic API Compatibility](https://www.navidrome.org/docs/developers/subsonic-api/)

## Screenshots

<p align="left">
    <img height="550" src="https://raw.githubusercontent.com/navidrome/navidrome/master/.github/screenshots/ss-mobile-login.png">
    <img height="550" src="https://raw.githubusercontent.com/navidrome/navidrome/master/.github/screenshots/ss-mobile-player.png">
    <img height="550" src="https://raw.githubusercontent.com/navidrome/navidrome/master/.github/screenshots/ss-mobile-album-view.png">
    <img width="550" src="https://raw.githubusercontent.com/navidrome/navidrome/master/.github/screenshots/ss-desktop-player.png">
</p>
