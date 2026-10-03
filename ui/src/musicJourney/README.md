# Music Journey

Music Journey turns the Navidrome library into a guided listening session:
the user picks a mood, duration, intensity, discovery level (and optionally
a genre/era), and the engine builds a phased sequence of tracks from _their
own library_ — Arrival → Flow → Peak → Drift — instead of a random shuffle.

## Architecture

```
UI (JourneyHome, Creator, NowPlaying, History)
 ↓
Hooks: useJourney (generate/adjust/stop/replay), useJourneyWatcher (player sync)
 ↓
Data: fetchSongPool (incremental slices via the existing dataProvider, cached)
 ↓
Engine (pure, dependency-free): journeyEngine → scoring + transitions + phases
                                + personalization + nameGenerator
 ↓
State: journeyReducer (active journey, history, skip feedback) + localStorage
 ↓
Playback: the EXISTING audio player — journeys fill the normal queue via
          playTracks; live re-routes use PLAYER_REBUILD_QUEUE, which keeps
          the current track's uuid so playback is never interrupted.
```

## Data used (all from Navidrome, nothing external)

- `bpm`, `genre`, `year`, `duration` → derived energy/valence/tempo profile
- `starred`, `rating`, `playCount`, `playDate` → personalization & freshness
- listening-derived genre/artist/decade affinity maps

## Key behaviours

- **Scoring**: weighted components (mood/genre/transition/preference/
  freshness/discovery) with configurable weights in `engine/weights.js`.
- **Transition intelligence**: energy/tempo cliffs, unrelated genre jumps
  and artist monotony are penalized; same-album continuity is rewarded.
- **Discovery slider**: flattens the top-K sampling temperature and flips
  the familiarity term — always inside the user's library.
- **Live steering**: "More energetic / relaxed / familiar / Surprise me /
  Less <genre>" rebuild only the remaining part; played tracks are kept.
- **Skip intelligence**: leaving a journey track before 60% records a skip
  event; similar content is temporarily down-weighted (bounded + decaying).
- **Reasons**: every pick carries an explanation assembled from the
  dominant computed components — never invented facts.
- **Graceful degradation**: each library slice is optional; small libraries
  produce shorter journeys; empty pools surface actionable error copy.

## Testing

Engine modules are pure functions and covered by colocated vitest suites
(`*.test.js`). The reducer is tested in isolation with a mocked store.
