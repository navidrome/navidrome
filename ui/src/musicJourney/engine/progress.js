/**
 * Derive the live position inside a journey from the player state.
 * Shared by the progress bar, the stage stepper and end-detection logic.
 */
import { buildNarrative, phaseIndexAt } from './phases'

/**
 * @param {import('./types').MusicJourney} journey
 * @param {Array} queue player queue items ({uuid, trackId, ...})
 * @param {Object} current player's current info ({uuid, trackId, currentTime, ended})
 */
export function computeJourneyPosition(journey, queue = [], current = {}) {
  const indexOfTrack = new Map()
  journey.tracks.forEach((t, i) => {
    if (!indexOfTrack.has(t.track.id)) indexOfTrack.set(t.track.id, i)
  })

  const currentIdx = current?.trackId
    ? (indexOfTrack.get(current.trackId) ?? -1)
    : -1

  let elapsed = 0
  for (let i = 0; i < currentIdx && i < journey.tracks.length; i++) {
    elapsed += journey.tracks[i].track.duration
  }
  if (currentIdx >= 0) {
    elapsed += Math.min(
      Number(current.currentTime) || 0,
      journey.tracks[currentIdx]?.track.duration || 0,
    )
  }

  const total = journey.estimatedDuration
  const narrative = buildNarrative(journey.preferences)
  const phaseIdx = phaseIndexAt(narrative, elapsed, total)

  const journeyInQueue = queue.some((item) => indexOfTrack.has(item?.trackId))

  return {
    currentIdx,
    elapsed,
    total,
    phaseIdx,
    narrative,
    current: currentIdx >= 0 ? journey.tracks[currentIdx] : null,
    next: journey.tracks[currentIdx + 1] || null,
    journeyInQueue,
    finished:
      Boolean(current?.ended) && currentIdx === journey.tracks.length - 1,
  }
}

/** Most frequent genre among the tracks that are still ahead. */
export function dominantRemainingGenre(journey, fromIndex) {
  const counts = new Map()
  for (const t of journey.tracks.slice(fromIndex)) {
    const g = String(t.track.genre || '')
      .trim()
      .toLowerCase()
    if (g) counts.set(g, (counts.get(g) || 0) + 1)
  }
  const sorted = [...counts.entries()].sort((a, b) => b[1] - a[1])
  return sorted.length ? sorted[0][0] : null
}
