# Session Notes — Outreach Drafts

## Posted

- **Draft 1** ("Show and tell" on `navidrome/navidrome`): live at
  [navidrome/navidrome#5781](https://github.com/navidrome/navidrome/discussions/5781).
- **Own repo Discussions set up**: welcome post pinned, Cirque beta-testing announcement live at
  [navidrome_experimental#15](https://github.com/RFLundgren/navidrome_experimental/discussions/15).

## Recommended approach

Posted **Draft 1 only** (see link above), to Discussions → "Show and tell" on `navidrome/navidrome`. That category
exists specifically for sharing personal builds/forks — low friction, opt-in for readers, doesn't interrupt anyone's
issue-tracking workflow. Still holding off on Draft 2 (a direct comment on issue #793) — an unsolicited comment on
someone else's tracked issue reads more presumptuous than a Discussions post, even with humble framing. Better to
let people find it via the Discussions post and link it into #793 themselves if they want to. Revisit Draft 2 later
if the Discussions post gets traction and it feels like a natural next step.

## Draft 1 — "Show and tell" post for navidrome/navidrome Discussions (POSTED — see link above)

**Title:** navidrome-experimental: a fork adding full Podcast support (Subsonic API) + physical folder browsing

> Hey all — I've been running a personal fork of Navidrome and wanted to share what I've added, in case it's useful to anyone else.
>
> **Podcast support** ([github.com/RFLundgren/navidrome_experimental](https://github.com/RFLundgren/navidrome_experimental)) — built through the real Subsonic API spec's podcast endpoints (not a Navidrome-only extension), so any client that's implemented that part of the spec gets full support with no server-specific hacks needed. (Whether your particular client has built that UI is up to its own developers — spec coverage for podcasts varies a lot across the ecosystem, so this isn't a guarantee your app already shows a podcasts tab.)
> - Discovery via search or live regional top-charts (manual RSS URL entry also supported)
> - Stream-only or download-to-disk, set per channel
> - Retention policies (episode count / age / total storage, oldest-downloaded-first cleanup)
> - Downloaded episodes can go into regular playlists alongside songs
> - Listened tracking, per user
> - Full spec coverage: `getPodcasts`, `getNewestPodcasts`, `createPodcastChannel`, `downloadPodcastEpisode`, streaming/download all work through the standard endpoints
>
> **Physical folder browsing** is also included — navigate your library the way it's laid out on disk, with recursive play/shuffle/playlist actions, ZIP downloads, and folder-pinned playlists.
>
> Screenshots of both in the [README](https://github.com/RFLundgren/navidrome_experimental#readme) if you want to see it before trying it.
>
> **To try it:** swap `ghcr.io/rflundgren/navidrome_experimental:stable` in for the official image in your `docker-compose.yml` — it tracks upstream closely and only adds tables/migrations, so your existing `/data` volume and library carry over untouched, `docker compose pull && docker compose up -d` is all it takes.
>
> If you'd rather not touch your existing instance at all, it's just as easy to run it side-by-side instead — spin up a second container from the same image, pointed at a different `/data` volume and a different host port (e.g. `4534:4533`), and it won't interfere with what's already running. Good way to poke around before deciding whether to switch your main setup over.
>
> Design writeup: [PODCAST_PLAN.md](https://github.com/RFLundgren/navidrome_experimental/blob/master/PODCAST_PLAN.md)
>
> Also in the works: dedicated clients ("Cirque") for Android and Windows desktop, currently in private testing. They won't be open source — planning to release them as donationware — but if you want to hear when public testing opens, there's a [thread for that](https://github.com/RFLundgren/navidrome_experimental/discussions/15) where I'll post updates.
>
> Happy to answer questions if anyone wants to try it, and open to feedback on the podcast design in particular — I know [#793](https://github.com/navidrome/navidrome/issues/793) has been open a long time, curious if this is close to what the project would want if there's ever appetite to bring something like it upstream.

## Draft 2 — comment on upstream issue #793 ("Implement podcast subsonic api")

> I built a working implementation of this in my fork, in case it's useful as reference (or a starting point): [github.com/RFLundgren/navidrome_experimental](https://github.com/RFLundgren/navidrome_experimental)
>
> Full Subsonic API coverage (`getPodcasts`, `getNewestPodcasts`, `createPodcastChannel`, `refreshPodcasts`, delete endpoints, `downloadPodcastEpisode`), stream-or-download per channel, retention policies, and playlist integration for downloaded episodes. Design notes: [PODCAST_PLAN.md](https://github.com/RFLundgren/navidrome_experimental/blob/master/PODCAST_PLAN.md)
>
> Docker image if anyone wants to try it: `ghcr.io/rflundgren/navidrome_experimental:stable` (drop-in for the official image — existing library/data carry over). Happy to discuss the approach or help adapt it if there's interest.

## Related, not yet drafted

- A similar comment for the folder-browsing discussions ([#2414](https://github.com/navidrome/navidrome/discussions/2414),
  [#3077](https://github.com/navidrome/navidrome/discussions/3077), [#2374](https://github.com/navidrome/navidrome/discussions/2374),
  [#3521](https://github.com/navidrome/navidrome/discussions/3521)) — lower engagement than #793, mentioned in
  passing in Draft 1 above but not written up as its own post yet.
