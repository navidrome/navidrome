import subsonic from '../subsonic'

// Number of peaks requested from the server. Bars are resampled from these to fit the bar width.
export const WAVEFORM_SAMPLES = 500

const BAR_WIDTH = 2
const BAR_GAP = 1
const UNPLAYED_ALPHA = 0.5
const MAX_CACHED = 50

const cache = new Map()

// loadWaveform fetches the peaks of a song, keeping the most recently used ones in memory so
// replaying or skipping back does not refetch them. Aborting signal cancels the request, which
// also stops the server from decoding a song that is no longer playing.
export const loadWaveform = (id, signal) => {
  if (cache.has(id)) {
    const cached = cache.get(id)
    cache.delete(id)
    cache.set(id, cached)
    return cached
  }
  const promise = subsonic
    .getWaveform(id, WAVEFORM_SAMPLES, { signal })
    .then((resp) => {
      const peaks = resp.json['subsonic-response']?.waveform?.peaks
      if (!peaks?.length) {
        throw new Error('No waveform available')
      }
      return peaks
    })
  const evict = () => {
    if (cache.get(id) === promise) {
      cache.delete(id)
    }
  }
  promise.catch(evict)
  // A request aborted while pending is evicted right away, so a call made before its rejection
  // settles (e.g. skipping away and straight back) starts a fresh request instead of reusing it.
  let settled = false
  promise.finally(() => (settled = true)).catch(() => {})
  signal?.addEventListener('abort', () => !settled && evict(), { once: true })
  cache.set(id, promise)
  if (cache.size > MAX_CACHED) {
    cache.delete(cache.keys().next().value)
  }
  return promise
}

export const clearWaveformCache = () => cache.clear()

// resample returns count values. The server sends RMS values, so each output is the quadratic
// mean of the values it covers. When count is larger than the number of peaks, peaks are repeated.
export const resample = (peaks, count) => {
  const out = new Array(count)
  for (let i = 0; i < count; i++) {
    const from = Math.floor((i * peaks.length) / count)
    const to = Math.max(from + 1, Math.floor(((i + 1) * peaks.length) / count))
    let sum = 0
    for (let j = from; j < to; j++) {
      sum += peaks[j] * peaks[j]
    }
    out[i] = Math.sqrt(sum / (to - from))
  }
  return out
}

// drawWaveform draws peaks as vertically centered bars. Heights are the squared RMS relative to the
// loudest peak (i.e. power), which spreads out the loud range most mastered music sits in. Bars
// left of progress (0-1) are drawn in played, the rest in unplayed at half opacity.
export const drawWaveform = (
  ctx,
  peaks,
  width,
  height,
  progress,
  played,
  unplayed,
) => {
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

  bars.forEach((peak, i) => {
    const x = i * step
    const h = Math.max(1, (peak / loudest) ** 2 * height)
    const isPlayed = x + BAR_WIDTH / 2 <= playedX
    ctx.fillStyle = isPlayed ? played : unplayed
    ctx.globalAlpha = isPlayed ? 1 : UNPLAYED_ALPHA
    ctx.fillRect(x, mid - h / 2, BAR_WIDTH, h)
  })
  ctx.globalAlpha = 1
}
