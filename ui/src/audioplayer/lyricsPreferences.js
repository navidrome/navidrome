// Local-only lyrics preferences. Storage can be unavailable (private mode,
// blocked third-party contexts), so every access degrades to session behavior.

const storage = () =>
  typeof window === 'undefined' ? null : window.localStorage

const LETTER_WAVE_STORAGE_KEY = 'lyrics.letterWave'

export const readLetterWavePreference = () => {
  try {
    return storage()?.getItem(LETTER_WAVE_STORAGE_KEY) === 'true'
  } catch {
    return false
  }
}

export const writeLetterWavePreference = (enabled) => {
  try {
    storage()?.setItem(LETTER_WAVE_STORAGE_KEY, enabled ? 'true' : 'false')
  } catch {
    // Session-only fallback.
  }
}
