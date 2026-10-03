/**
 * Journey title generator — fully local, deterministic for a given seed.
 * Titles are assembled from word banks keyed by mood, energy and time of
 * day so the name reflects the journey's actual parameters.
 */
import { hashSeed, mulberry32 } from './types'

const ADJECTIVES = {
  chill: ['Quiet', 'Velvet', 'Slow', 'Gentle', 'Weightless', 'Amber', 'Hushed'],
  focus: ['Steady', 'Clear', 'Glass', 'Quiet', 'Lucid', 'Deep', 'Still'],
  energy: [
    'Electric',
    'Neon',
    'Burning',
    'Wild',
    'Restless',
    'Charged',
    'Bright',
  ],
  happy: ['Golden', 'Sunny', 'Open', 'Glowing', 'Vivid', 'Warm', 'Easy'],
  melancholic: [
    'Grey',
    'Fading',
    'Distant',
    'Rainy',
    'Faint',
    'Blue',
    'Lonely',
  ],
}

const NOUNS = {
  chill: ['Drift', 'Horizon', 'Tide', 'Window', 'Morning', 'Harbour', 'Breath'],
  focus: [
    'Current',
    'Signal',
    'Passage',
    'Meridian',
    'Study',
    'Field',
    'Orbit',
  ],
  energy: [
    'Drive',
    'Engine',
    'Skyline',
    'Circuit',
    'Runway',
    'Voltage',
    'Pulse',
  ],
  happy: [
    'Afternoon',
    'Parade',
    'Carousel',
    'Picnic',
    'Boulevard',
    'Refrain',
    'Daylight',
  ],
  melancholic: [
    'Rain',
    'Echo',
    'Winter',
    'Letter',
    'Station',
    'Ember',
    'Silence',
  ],
}

const TIME_PREFIX = {
  morning: ['Sunrise', 'Early'],
  afternoon: ['Golden Hour', 'Midday'],
  evening: ['Sunset', 'Dusk'],
  night: ['Midnight', 'Late Night'],
}

const timeOfDay = (date) => {
  const h = date.getHours()
  if (h < 6) return 'night'
  if (h < 12) return 'morning'
  if (h < 18) return 'afternoon'
  if (h < 23) return 'evening'
  return 'night'
}

/**
 * @param {import('./types').JourneyPreferences} prefs
 * @param {Date} [date]
 * @param {number} [variant] bump to get a different name for same params
 */
export function generateJourneyName(prefs, date = new Date(), variant = 0) {
  const mood = ADJECTIVES[prefs.mood] ? prefs.mood : 'chill'
  const seed = hashSeed(
    `${prefs.mood}|${prefs.durationMinutes}|${Math.round(prefs.intensity * 10)}|` +
      `${Math.round(prefs.discovery * 10)}|${prefs.genre || ''}|${prefs.decade || ''}|${variant}`,
  )
  const rnd = mulberry32(seed)

  const era = prefs.decade ? ` '${String(prefs.decade).slice(2, 4)}s` : ''
  const tod = timeOfDay(date)

  // A third of the time lead with a time-of-day flavour ("Midnight Drive").
  if (rnd() < 0.34) {
    const prefixes = TIME_PREFIX[tod]
    const prefix = prefixes[Math.floor(rnd() * prefixes.length)]
    const noun = NOUNS[mood][Math.floor(rnd() * NOUNS[mood].length)]
    return `${prefix} ${noun}${era}`
  }

  const adj = ADJECTIVES[mood][Math.floor(rnd() * ADJECTIVES[mood].length)]
  const noun = NOUNS[mood][Math.floor(rnd() * NOUNS[mood].length)]
  return `${adj} ${noun}${era}`
}
