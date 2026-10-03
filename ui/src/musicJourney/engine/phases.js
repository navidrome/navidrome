/**
 * Journey narrative phases.
 *
 * Each mood archetype has its own arc (a chill journey is
 * Arrival -> Flow -> Deep Focus -> Drift, an energetic one is
 * Warm Up -> Build -> Peak -> Afterglow). The number of phases adapts to
 * the requested duration so a 10 minute journey is not sliced into five
 * microscopic chapters.
 */
import { MOODS } from './types'

export const PHASE_ARCHETYPES = {
  chill: [
    { id: 'arrival', name: 'Arrival', weight: 0.2, energyBias: 0.75 },
    { id: 'flow', name: 'Flow', weight: 0.3, energyBias: 1.0 },
    { id: 'deep-focus', name: 'Deep Focus', weight: 0.3, energyBias: 1.08 },
    { id: 'drift', name: 'Drift', weight: 0.2, energyBias: 0.8 },
  ],
  focus: [
    { id: 'settle', name: 'Settle In', weight: 0.2, energyBias: 0.8 },
    { id: 'deep-focus', name: 'Deep Focus', weight: 0.35, energyBias: 0.95 },
    { id: 'sustain', name: 'Sustain', weight: 0.3, energyBias: 1.05 },
    { id: 'fade-out', name: 'Fade Out', weight: 0.15, energyBias: 0.7 },
  ],
  energy: [
    { id: 'warm-up', name: 'Warm Up', weight: 0.2, energyBias: 0.8 },
    { id: 'build', name: 'Build', weight: 0.25, energyBias: 1.0 },
    { id: 'peak', name: 'Peak', weight: 0.3, energyBias: 1.25 },
    { id: 'afterglow', name: 'Afterglow', weight: 0.25, energyBias: 0.85 },
  ],
  happy: [
    { id: 'sunrise', name: 'Sunrise', weight: 0.25, energyBias: 0.85 },
    { id: 'groove', name: 'Groove', weight: 0.3, energyBias: 1.05 },
    { id: 'high-spirits', name: 'High Spirits', weight: 0.25, energyBias: 1.2 },
    { id: 'golden-hour', name: 'Golden Hour', weight: 0.2, energyBias: 0.9 },
  ],
  melancholic: [
    { id: 'first-light', name: 'First Light', weight: 0.25, energyBias: 0.75 },
    { id: 'reflection', name: 'Reflection', weight: 0.3, energyBias: 0.95 },
    { id: 'immersion', name: 'Immersion', weight: 0.25, energyBias: 1.05 },
    { id: 'release', name: 'Release', weight: 0.2, energyBias: 0.85 },
  ],
  // Generic arc used when a custom/unknown mood is requested.
  default: [
    { id: 'intro', name: 'Intro', weight: 0.15, energyBias: 0.8 },
    { id: 'build', name: 'Build', weight: 0.25, energyBias: 1.0 },
    { id: 'peak', name: 'Peak', weight: 0.3, energyBias: 1.2 },
    { id: 'cooldown', name: 'Cooldown', weight: 0.2, energyBias: 0.9 },
    { id: 'outro', name: 'Outro', weight: 0.1, energyBias: 0.7 },
  ],
}

/** Mood presets: base energy/valence targets before intensity scaling. */
export const MOOD_PRESETS = {
  chill: { energy: 0.3, valence: 0.55, archetype: 'chill' },
  focus: { energy: 0.25, valence: 0.5, archetype: 'focus' },
  energy: { energy: 0.75, valence: 0.62, archetype: 'energy' },
  happy: { energy: 0.6, valence: 0.85, archetype: 'happy' },
  melancholic: { energy: 0.32, valence: 0.25, archetype: 'melancholic' },
}

/** Phases scale down for short journeys to keep each one meaningful. */
const phaseCountForDuration = (minutes) =>
  minutes < 15 ? 2 : minutes < 30 ? 3 : 4

/**
 * Base energy target for a journey: mostly driven by the intensity slider,
 * nudged toward the mood's natural range so "chill + high intensity" still
 * feels chillier than "energy + low intensity".
 */
export const baseEnergyOf = (prefs) => {
  const preset = MOOD_PRESETS[prefs.mood] || MOOD_PRESETS.chill
  return Math.min(1, Math.max(0, preset.energy * 0.35 + prefs.intensity * 0.65))
}

/**
 * Expand the archetype into the concrete narrative for this journey,
 * including per-phase energy/valence targets.
 *
 * @param {import('./types').JourneyPreferences} prefs
 * @returns {Array<import('./types').JourneyStage & {targetEnergy: number, targetValence: number|null}>}
 */
export function buildNarrative(prefs) {
  const preset = MOOD_PRESETS[prefs.mood] || {
    energy: 0.5,
    valence: null,
    archetype: 'default',
  }
  const archetype =
    PHASE_ARCHETYPES[preset.archetype] || PHASE_ARCHETYPES.default
  const count = Math.min(
    phaseCountForDuration(prefs.durationMinutes),
    archetype.length,
  )
  // Keep the narrative's beginning and end, drop middle phases when shortening.
  let phases = archetype
  if (count < archetype.length) {
    phases = [archetype[0]]
    const middle = archetype.slice(1, -1)
    while (phases.length < count - 1 && middle.length) {
      phases.push(middle.splice(Math.floor(middle.length / 2), 1)[0])
    }
    phases.push(archetype[archetype.length - 1])
  }

  const weightSum = phases.reduce((s, p) => s + p.weight, 0)
  const base = baseEnergyOf(prefs)
  return phases.map((p) => ({
    ...p,
    weight: p.weight / weightSum,
    targetEnergy: Math.min(1, Math.max(0.05, base * p.energyBias)),
    targetValence: preset.valence,
  }))
}

/**
 * Allocate journey duration across phases. Guarantees the weights sum to 1
 * and every phase gets a positive share.
 */
export function allocatePhaseDurations(narrative, totalSeconds) {
  return narrative.map((phase) => ({
    phase,
    seconds: totalSeconds * phase.weight,
  }))
}

/** Index of the phase a given elapsed time falls into. */
export function phaseIndexAt(narrative, elapsedSeconds, totalSeconds) {
  if (!narrative.length) return 0
  let acc = 0
  for (let i = 0; i < narrative.length; i++) {
    acc += narrative[i].weight * totalSeconds
    if (elapsedSeconds < acc) return i
  }
  return narrative.length - 1
}

export const moodExists = (mood) => MOODS.includes(mood)
