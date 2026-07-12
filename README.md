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

**navidrome-experimental** is a personal fork of [Navidrome](https://github.com/navidrome/navidrome) adding features
not yet available upstream. It tracks upstream closely and aims to stay compatible with the standard Navidrome/Subsonic
ecosystem (clients, plugins, themes) — it just adds a few things on top:

- **[Podcast support](PODCAST_PLAN.md)** — subscribe by search or regional top-charts, stream or download episodes,
  per-channel retention policies, downloaded episodes in regular playlists, full Subsonic API coverage (works with
  any Subsonic-compatible client, not just the web UI).
- **[Physical folder browsing](navidrome-folder-roadmap.md)** — navigate your library exactly as it's laid out on
  disk, with recursive play/shuffle/playlist actions, ZIP downloads, folder-pinned playlists, and Subsonic client
  compatibility. See below for details.

Docker images are published to `ghcr.io/rflundgren/navidrome_experimental`. Everything else — installation,
configuration, the Subsonic API, plugins — works exactly like upstream Navidrome; see the
[Documentation](#documentation) section below.

**Note**: The `master` branch may be in an unstable or even broken state during development. 
Please use [releases](https://github.com/navidrome/navidrome/releases) instead of 
the `master` branch in order to get a stable set of binaries.

## [Check out our Live Demo!](https://www.navidrome.org/demo/)

__Any feedback is welcome!__ If you need/want a new feature, find a bug or think of any way to improve Navidrome, 
please file a [GitHub issue](https://github.com/navidrome/navidrome/issues) or join the discussion in our 
[Subreddit](https://www.reddit.com/r/navidrome/). If you want to contribute to the project in any other way 
([ui/backend dev](https://www.navidrome.org/docs/developers/), 
[translations](https://www.navidrome.org/docs/developers/translations/), 
[themes](https://www.navidrome.org/docs/developers/creating-themes)), please join the chat in our 
[Discord server](https://discord.gg/xh7j7yF). 

## Installation

See instructions on the [project's website](https://www.navidrome.org/docs/installation/)

## Cloud Hosting

[PikaPods](https://www.pikapods.com) has partnered with us to offer you an 
[officially supported, cloud-hosted solution](https://www.navidrome.org/docs/installation/managed/#pikapods). 
A share of the revenue helps fund the development of Navidrome at no additional cost for you.

[![PikaPods](https://www.pikapods.com/static/run-button.svg)](https://www.pikapods.com/pods?run=navidrome)

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
 - **Themeable**, modern and responsive **Web interface** based on [Material UI](https://material-ui.com)
 - **Compatible** with all Subsonic/Madsonic/Airsonic [clients](https://www.navidrome.org/docs/overview/#apps)
 - **Transcoding** on the fly. Can be set per user/player. **Opus encoding is supported**
 - Translated to **various languages**

## Podcast Support (Experimental)

This version of Navidrome includes full **Podcast support** over RSS, built specifically to work through the real
Subsonic API — not just the web UI — so any Subsonic-compatible client can subscribe, download, and stream episodes
exactly like it would for a standard Subsonic server.

### Current Features
- **Discovery**: subscribe by searching iTunes' podcast directory, or pick from live, region-specific top charts.
- **Stream or download**: per-channel policy — stream-only (proxied through the server on demand) or auto-download
  new/all episodes to disk.
- **Retention policies**: per-channel limits on episode count, age, or total storage, with oldest-downloaded-first
  cleanup.
- **Playlists**: downloaded episodes can be added to regular playlists alongside songs, reordered, and exported.
- **Listened tracking**: episodes you've played are marked, per user.
- **Full Subsonic API coverage**: `getPodcasts`, `getNewestPodcasts`, `createPodcastChannel`,
  `downloadPodcastEpisode`, and streaming/download both work through the standard endpoints, so third-party apps
  need no special support.
- **Personal toggle**: hide the Podcasts section from your own sidebar if you don't use it (same as the Folder
  view toggle below).

For more details, including what's still on the roadmap, see [PODCAST_PLAN.md](PODCAST_PLAN.md).

## Physical Folder Browsing (Experimental)

This version of Navidrome includes a major new feature: **Physical Folder Browsing**. This allows you to navigate your music library exactly as it is organized on your hard drive, bypassing traditional metadata-based views.

### Current Features
- **Hierarchical Navigation**: Browse through folders and subfolders with functional breadcrumbs.
- **Recursive Actions**: Play All, Shuffle, or Add to Playlist for an entire folder hierarchy with one click.
- **ZIP Downloads**: Download entire physical folders as a ZIP archive directly from the UI.
- **Scoped Search**: Search for specific tracks or subfolders directly within a physical folder hierarchy.
- **Visual Polish**: Support for folder thumbnails (including composite artwork), a Grid/List view toggle, and automatic hiding of empty UI sections.
- **"Show in Folder"**: Jump directly to a song or album's physical location from anywhere in the app.
- **Folder Sync**: "Pin" physical folders as virtual Navidrome playlists that automatically update during library scans.
- **Subsonic Integration**: Compatible with mobile apps that support physical folder browsing.

For more details, see [navidrome-folder-roadmap.md](navidrome-folder-roadmap.md).

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
