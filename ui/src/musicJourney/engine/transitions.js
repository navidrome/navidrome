/**
 * Transition intelligence.
 *
 * `transitionScore(prev, next, prefs)` is the standalone judgement of how
 * well one track can follow another. The scoring module folds it into the
 * total via the transition weight; keeping it separate lets us test the
 * transition logic in isolation and lets future strategy providers (e.g. an
 * ML similarity model) implement the same contract.
 */
import { JOURNEY_PENALTIES, maxEnergyStep } from './weights'

const genreGroup = (genre) => {
  const g = String(genre || '').toLowerCase()
  if (!g) return null
  const groups = [
    [
      'electronic',
      'house',
      'techno',
      'trance',
      'edm',
      'dubstep',
      'drum and bass',
      'synthwave',
      'electro',
    ],
    [
      'rock',
      'metal',
      'punk',
      'alternative',
      'grunge',
      'hardcore',
      'indie rock',
    ],
    ['pop', 'disco', 'dance', 'funk'],
    ['hip hop', 'hip-hop', 'rap', 'trap', 'r&b', 'rnb', 'soul'],
    ['jazz', 'blues', 'bossa nova', 'swing'],
    ['classical', 'orchestral', 'ambient', 'soundtrack', 'score', 'drone'],
    ['folk', 'acoustic', 'country', 'bluegrass', 'singer-songwriter'],
    ['reggae', 'ska', 'latin', 'dub'],
    ['lofi', 'lo-fi', 'chill', 'chillout', 'downtempo', 'lounge'],
  ]
  for (let i = 0; i < groups.length; i++) {
    if (groups[i].some((k) => g.includes(k))) return i
  }
  return null
}

/**
 * @param {Object} prev  {song, profile} of the outgoing track
 * @param {Object} next  {song, profile} of the incoming track
 * @param {import('./types').JourneyPreferences} prefs
 * @returns {{score: number, penalties: string[]}} score 0..1 (1 = seamless)
 */
export function transitionScore(prev, next, prefs) {
  if (!prev || !prev.profile) return { score: 1, penalties: [] }
  const a = prev.profile
  const b = next.profile
  const penalties = []
  let score = 1

  // 1. Energy cliff — the classic "ambient -> aggressive metal" problem.
  const energyDelta = Math.abs(a.energy - b.energy)
  const allowed = maxEnergyStep(prefs.discovery)
  if (energyDelta > allowed) {
    score -= JOURNEY_PENALTIES.energyCliff * ((energyDelta - allowed) / allowed)
    penalties.push('energy-cliff')
  } else {
    // Reward gentle motion: small steps slightly preferred over flat lines.
    score -= energyDelta * 0.08
  }

  // 2. Tempo jump (only when both sides have real BPM evidence).
  if (a.bpm && b.bpm) {
    const ratio = Math.max(a.bpm, b.bpm) / Math.min(a.bpm, b.bpm)
    if (ratio > 1.5) {
      score -= 0.12
      penalties.push('tempo-jump')
    }
  }

  // 3. Genre jump between unrelated families (discovery softens this).
  const ga = genreGroup(prev.song?.genre)
  const gb = genreGroup(next.song?.genre)
  if (ga != null && gb != null && ga !== gb) {
    score -= 0.08 * (1 - prefs.discovery)
    penalties.push('genre-jump')
  }

  // 4. Mood clash (valence whiplash).
  if (Math.abs(a.valence - b.valence) > 0.45 && energyDelta > 0.3) {
    score -= 0.1
    penalties.push('mood-clash')
  }

  // 5. Artist/album continuity: a second track from the same album flows
  //    naturally, but a third artist-in-a-row is monotonous.
  if (prev.song && next.song) {
    if (next.song.albumId && next.song.albumId === prev.song.albumId) {
      score += 0.05
    } else if (
      next.song.artistId &&
      prev.song.artistId &&
      next.song.artistId === prev.song.artistId
    ) {
      score -= JOURNEY_PENALTIES.consecutiveArtist
      penalties.push('same-artist')
    }
  }

  return { score: Math.min(1, Math.max(0, score)), penalties }
}
