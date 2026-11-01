import subsonic from '../subsonic'
import {
  clearWaveformCache,
  drawWaveform,
  loadWaveform,
  resample,
  WAVEFORM_SAMPLES,
} from './waveform'

vi.mock('../subsonic', () => ({
  default: { getWaveform: vi.fn() },
}))

const waveformResponse = (peaks) => ({
  json: { 'subsonic-response': { status: 'ok', waveform: { peaks } } },
})

describe('resample', () => {
  it('returns the quadratic mean of each group when downsampling', () => {
    expect(resample([3, 4, 5, 5, 0, 0], 3)).toEqual([Math.sqrt(12.5), 5, 0])
  })

  it('repeats peaks when upsampling', () => {
    expect(resample([1, 5], 4)).toEqual([1, 1, 5, 5])
  })
})

describe('drawWaveform', () => {
  const fakeContext = () => {
    const ctx = { rects: [], globalAlpha: 1 }
    ctx.clearRect = vi.fn()
    ctx.fillRect = vi.fn((x, y, w, h) =>
      ctx.rects.push({
        x,
        y,
        w,
        h,
        color: ctx.fillStyle,
        alpha: ctx.globalAlpha,
      }),
    )
    return ctx
  }

  it('draws one centered bar per 3px, as power relative to the loudest peak', () => {
    const ctx = fakeContext()
    drawWaveform(ctx, [50, 100], 6, 20, 0, 'red', 'grey')
    expect(ctx.rects).toEqual([
      { x: 0, y: 7.5, w: 2, h: 5, color: 'grey', alpha: 0.5 },
      { x: 3, y: 0, w: 2, h: 20, color: 'grey', alpha: 0.5 },
    ])
  })

  it('draws played bars opaque in the played color', () => {
    const ctx = fakeContext()
    drawWaveform(ctx, [100, 100], 6, 20, 0.5, 'red', 'grey')
    expect(ctx.rects.map((r) => [r.color, r.alpha])).toEqual([
      ['red', 1],
      ['grey', 0.5],
    ])
    expect(ctx.globalAlpha).toBe(1)
  })

  it('draws silence as a flat line', () => {
    const ctx = fakeContext()
    drawWaveform(ctx, [0, 0], 6, 20, 0, 'red', 'grey')
    expect(ctx.rects.map((r) => r.h)).toEqual([1, 1])
  })

  it('draws nothing without peaks', () => {
    const ctx = fakeContext()
    drawWaveform(ctx, [], 6, 20, 0, 'red', 'grey')
    expect(ctx.fillRect).not.toHaveBeenCalled()
  })
})

describe('loadWaveform', () => {
  beforeEach(() => {
    clearWaveformCache()
    vi.clearAllMocks()
  })

  it('fetches the peaks of a song', async () => {
    subsonic.getWaveform.mockResolvedValue(waveformResponse([1, 2, 3]))
    await expect(loadWaveform('song-1')).resolves.toEqual([1, 2, 3])
    expect(subsonic.getWaveform).toHaveBeenCalledWith(
      'song-1',
      WAVEFORM_SAMPLES,
      { signal: undefined },
    )
  })

  it('passes the abort signal to the request', async () => {
    subsonic.getWaveform.mockResolvedValue(waveformResponse([1]))
    const controller = new AbortController()
    await loadWaveform('song-1', controller.signal)
    expect(subsonic.getWaveform).toHaveBeenCalledWith(
      'song-1',
      WAVEFORM_SAMPLES,
      { signal: controller.signal },
    )
  })

  it('does not reuse a pending request once it is aborted', async () => {
    subsonic.getWaveform.mockReturnValueOnce(new Promise(() => {}))
    const controller = new AbortController()
    loadWaveform('song-1', controller.signal).catch(() => {})
    controller.abort()

    subsonic.getWaveform.mockResolvedValueOnce(waveformResponse([4]))
    await expect(loadWaveform('song-1')).resolves.toEqual([4])
    expect(subsonic.getWaveform).toHaveBeenCalledTimes(2)
  })

  it('keeps fetched peaks when a later abort fires', async () => {
    subsonic.getWaveform.mockResolvedValueOnce(waveformResponse([1]))
    const controller = new AbortController()
    await loadWaveform('song-1', controller.signal)
    controller.abort()

    await expect(loadWaveform('song-1')).resolves.toEqual([1])
    expect(subsonic.getWaveform).toHaveBeenCalledTimes(1)
  })

  it('refetches a song whose request was aborted', async () => {
    subsonic.getWaveform.mockRejectedValueOnce(
      new DOMException('Aborted', 'AbortError'),
    )
    await expect(loadWaveform('song-1')).rejects.toThrow()

    subsonic.getWaveform.mockResolvedValueOnce(waveformResponse([4]))
    await expect(loadWaveform('song-1')).resolves.toEqual([4])
    expect(subsonic.getWaveform).toHaveBeenCalledTimes(2)
  })

  it('reuses previously fetched peaks', async () => {
    subsonic.getWaveform.mockResolvedValue(waveformResponse([1, 2, 3]))
    await loadWaveform('song-1')
    await loadWaveform('song-1')
    expect(subsonic.getWaveform).toHaveBeenCalledTimes(1)
  })

  it('rejects and retries later when the server has no waveform', async () => {
    subsonic.getWaveform.mockResolvedValueOnce({
      json: { 'subsonic-response': { status: 'failed' } },
    })
    await expect(loadWaveform('song-1')).rejects.toThrow()

    subsonic.getWaveform.mockResolvedValueOnce(waveformResponse([4]))
    await expect(loadWaveform('song-1')).resolves.toEqual([4])
  })
})
