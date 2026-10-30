import subsonic from '../subsonic'

// Number of peaks requested from the server. Bars are resampled from these to fit the bar width.
export const WAVEFORM_SAMPLES = 500

const BAR_WIDTH = 2
const BAR_GAP = 1
const UNPLAYED_ALPHA = 0.35
const MAX_CACHED = 50

const cache = new Map()

// loadWaveform fetches the peaks of a song, keeping the most recently used ones in memory so
// replaying or skipping back does not refetch them.
export const loadWaveform = (id) => {
  if (cache.has(id)) {
    const cached = cache.get(id)
    cache.delete(id)
    cache.set(id, cached)
    return cached
  }
  const promise = subsonic.getWaveform(id, WAVEFORM_SAMPLES).then((resp) => {
    const peaks = resp.json['subsonic-response']?.waveform?.peaks
    if (!peaks?.length) {
      throw new Error('No waveform available')
    }
    return peaks
  })
  promise.catch(() => cache.delete(id))
  cache.set(id, promise)
  if (cache.size > MAX_CACHED) {
    cache.delete(cache.keys().next().value)
  }
  return promise
}

export const clearWaveformCache = () => cache.clear()

// resample returns count values, each the maximum of the peaks it covers. When count is larger
// than the number of peaks, peaks are repeated.
export const resample = (peaks, count) => {
  const out = new Array(count)
  for (let i = 0; i < count; i++) {
    const from = Math.floor((i * peaks.length) / count)
    const to = Math.max(from + 1, Math.floor(((i + 1) * peaks.length) / count))
    let max = 0
    for (let j = from; j < to; j++) {
      max = Math.max(max, peaks[j])
    }
    out[i] = max
  }
  return out
}

// drawWaveform draws peaks as vertically centered bars, scaled so the loudest peak fills the
// height. Bars left of progress (0-1) are drawn opaque, the rest translucent.
export const drawWaveform = (ctx, peaks, width, height, progress, color) => {
  ctx.clearRect(0, 0, width, height)
  if (!peaks?.length || width <= 0 || height <= 0) {
    return
  }
  const step = BAR_WIDTH + BAR_GAP
  const count = Math.max(1, Math.floor((width + BAR_GAP) / step))
  const bars = resample(peaks, count)
  const loudest = Math.max(1, ...peaks)
  const playedX = progress * width
  const mid = height / 2

  ctx.fillStyle = color
  bars.forEach((peak, i) => {
    const x = i * step
    const h = Math.max(1, (peak / loudest) * height)
    ctx.globalAlpha = x + BAR_WIDTH / 2 <= playedX ? 1 : UNPLAYED_ALPHA
    ctx.fillRect(x, mid - h / 2, BAR_WIDTH, h)
  })
  ctx.globalAlpha = 1
}
