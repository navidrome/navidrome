# Session Notes — Outreach Drafts (on hold)

Drafted, **not yet posted**. Holding these until a few open questions below are resolved, then revisit.

## Open questions to resolve first

- Whether to scrub the `Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>` trailers currently present in
  a lot of the git history before pointing more outside attention at the repo (this would mean rewriting published
  history and force-pushing — needs a deliberate decision, not something to do casually).
- Anything else that comes up before posting publicly.

## Draft 1 — "Show and tell" post for navidrome/navidrome Discussions

**Title:** navidrome-experimental: a fork adding full Podcast support (Subsonic API) + physical folder browsing

> Hey all — I've been running a personal fork of Navidrome and wanted to share what I've added, in case it's useful to anyone else.
>
> **Podcast support** ([github.com/RFLundgren/navidrome_experimental](https://github.com/RFLundgren/navidrome_experimental)) — built specifically to work through the real Subsonic API, not just the web UI, so any Subsonic-compatible client can subscribe, download, and stream episodes exactly like a standard Subsonic server:
> - Discovery via search or live regional top-charts (manual RSS URL entry also supported)
> - Stream-only or download-to-disk, set per channel
> - Retention policies (episode count / age / total storage, oldest-downloaded-first cleanup)
> - Downloaded episodes can go into regular playlists alongside songs
> - Listened tracking, per user
> - Full spec coverage: `getPodcasts`, `getNewestPodcasts`, `createPodcastChannel`, `downloadPodcastEpisode`, streaming/download all work through the standard endpoints
>
> **Physical folder browsing** is also included — navigate your library the way it's laid out on disk, with recursive play/shuffle/playlist actions, ZIP downloads, and folder-pinned playlists.
>
> **To try it:** swap `ghcr.io/rflundgren/navidrome_experimental:develop` in for the official image in your `docker-compose.yml` — it tracks upstream closely and only adds tables/migrations, so your existing `/data` volume and library carry over untouched, `docker compose pull && docker compose up -d` is all it takes.
>
> Design writeup: [PODCAST_PLAN.md](https://github.com/RFLundgren/navidrome_experimental/blob/master/PODCAST_PLAN.md)
>
> Happy to answer questions if anyone wants to try it, and open to feedback on the podcast design in particular — I know [#793](https://github.com/navidrome/navidrome/issues/793) has been open a long time, curious if this is close to what the project would want if there's ever appetite to bring something like it upstream.

## Draft 2 — comment on upstream issue #793 ("Implement podcast subsonic api")

> I built a working implementation of this in my fork, in case it's useful as reference (or a starting point): [github.com/RFLundgren/navidrome_experimental](https://github.com/RFLundgren/navidrome_experimental)
>
> Full Subsonic API coverage (`getPodcasts`, `getNewestPodcasts`, `createPodcastChannel`, `refreshPodcasts`, delete endpoints, `downloadPodcastEpisode`), stream-or-download per channel, retention policies, and playlist integration for downloaded episodes. Design notes: [PODCAST_PLAN.md](https://github.com/RFLundgren/navidrome_experimental/blob/master/PODCAST_PLAN.md)
>
> Docker image if anyone wants to try it: `ghcr.io/rflundgren/navidrome_experimental:develop` (drop-in for the official image — existing library/data carry over). Happy to discuss the approach or help adapt it if there's interest.

## Related, not yet drafted

- A similar comment for the folder-browsing discussions ([#2414](https://github.com/navidrome/navidrome/discussions/2414),
  [#3077](https://github.com/navidrome/navidrome/discussions/3077), [#2374](https://github.com/navidrome/navidrome/discussions/2374),
  [#3521](https://github.com/navidrome/navidrome/discussions/3521)) — lower engagement than #793, mentioned in
  passing in Draft 1 above but not written up as its own post yet.
