/**
 * Scoring weights for the Music Journey engine.
 *
 * These are deliberately kept in one configurable object: tuning the feel of
 * generated journeys should not require touching the scoring code itself.
 * When a future ML/LLM strategy provider is plugged in, it can simply
 * replace (or reweigh) these factors.
 */
export const JOURNEY_WEIGHTS = {
  /** How close the track's derived energy/valence is to the current phase target. */
  mood: 0.25,
  /** Affinity with the genres the user actually listens to. */
  genre: 0.15,
  /** Smoothness of the transition from the previous track. */
  transition: 0.2,
  /** Favorites, ratings and play counts. */
  userPreference: 0.15,
  /** Penalty for tracks played very recently (anti-repetition). */
  freshness: 0.1,
  /** Extra term whose direction is driven by the discovery slider. */
  discovery: 0.1,
  /** Affinity with artists the user plays often. */
  artist: 0.05,
}

/** Penalties are applied on top of the weighted score. */
export const JOURNEY_PENALTIES = {
  /** Same artist as the previous track. */
  consecutiveArtist: 0.12,
  /** Third (or more) track from the same artist in one journey. */
  artistSaturation: 0.2,
  /** Same album back-to-back beyond the first two tracks. */
  albumChain: 0.08,
  /** Energy jump between consecutive tracks larger than the allowed step. */
  energyCliff: 0.25,
}

/**
 * The maximum energy difference allowed between consecutive tracks before
 * the cliff penalty kicks in. Relaxed as discovery increases — a "surprise
 * me" journey tolerates sharper turns.
 */
export const maxEnergyStep = (discovery = 0.3) => 0.35 + discovery * 0.35

/** Tunables for the "why this track?" explanations. */
export const REASON_TOP_CONTRIBUTORS = 2
