import { act } from '@testing-library/react'
import { renderHook } from '@testing-library/react-hooks'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  KARAOKE_CHARACTER_CREST_RATIO,
  KARAOKE_CHARACTER_LIFT_EM,
  KARAOKE_CHARACTER_SETTLE_EM,
  KARAOKE_CHARACTER_WAVE_MS,
  KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS,
  KARAOKE_CHARACTER_WAVE_SPAN_RATIO,
  KARAOKE_CLOCK_DRIFT_RESET_MS,
  KARAOKE_HIGHLIGHT_LEAD_MS,
  KARAOKE_LINE_RELEASE_MS,
} from './lyricsKaraokeConstants'
import useLyricsTimeline from './useLyricsTimeline'

const createAudio = ({
  currentTime = 0,
  duration = 10,
  paused = true,
  playbackRate = 1,
} = {}) => {
  const target = new EventTarget()
  target.currentTime = currentTime
  target.duration = duration
  target.paused = paused
  target.playbackRate = playbackRate
  target.seeking = false
  return target
}

const lines = [
  {
    start: 0,
    end: 1000,
    tokens: [
      { start: 0, end: 500, value: 'first' },
      { start: 500, end: 1000, value: 'second' },
    ],
  },
  { start: 1000, end: 2000, tokens: [] },
]

const animationsOf = (node) => node?.__animations || []

const sweepAnimationOf = (tokenNode) =>
  animationsOf(
    tokenNode.querySelector('[data-lyrics-sweep-layer="true"]'),
  ).find((animation) =>
    animation.keyframes.some((frame) => frame.maskPosition != null),
  )

const characterAnimationsOf = (tokenNode) =>
  Array.from(tokenNode.querySelectorAll('[data-lyrics-character="true"]')).map(
    (node) =>
      animationsOf(node).find((animation) =>
        animation.keyframes.some((frame) => frame.transform != null),
      ),
  )

const createTokenNode = (text = '') => {
  const tokenNode = document.createElement('span')
  const sweepNode = document.createElement('span')
  sweepNode.dataset.lyricsSweepLayer = 'true'
  Array.from(text).forEach((character) => {
    const node = document.createElement('span')
    node.dataset.lyricsCharacter = 'true'
    node.textContent = character
    sweepNode.appendChild(node)
  })
  tokenNode.appendChild(sweepNode)
  return tokenNode
}

const registerToken = (result, key, window, text = '') => {
  const tokenNode = createTokenNode(text)
  act(() => {
    result.current.registerToken(key, { lineIndex: 0, window }, tokenNode)
  })
  return tokenNode
}

const syncNow = (result, time) => {
  act(() => {
    result.current.syncNow(time, true)
  })
}

const renderTimeline = ({
  audio = createAudio(),
  sourceLines = lines,
  reducedMotion = false,
} = {}) =>
  renderHook(() =>
    useLyricsTimeline({
      lines: sourceLines,
      audioInstance: audio,
      visible: true,
      reducedMotion,
    }),
  )

describe('useLyricsTimeline', () => {
  beforeEach(() => {
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation(() => 1)
    vi.spyOn(window, 'cancelAnimationFrame').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('synchronizes paused media without starting an animation loop', () => {
    const audio = createAudio({ currentTime: 0.25, paused: true })
    const { result } = renderTimeline({ audio })
    const lineNode = document.createElement('div')
    const tokenNode = createTokenNode()

    act(() => {
      result.current.registerLine(0, lineNode)
      result.current.registerToken(
        '0:main:0',
        {
          lineIndex: 0,
          window: { start: 0, end: 500 },
        },
        tokenNode,
      )
    })

    expect(result.current.activeIndexes).toEqual([0])
    expect(lineNode.dataset.active).toBe('true')
    expect(tokenNode.dataset.lyricsState).toBe('active')
    expect(sweepAnimationOf(tokenNode).currentTime).toBe(370)
    expect(sweepAnimationOf(tokenNode).playState).toBe('paused')
    expect(window.requestAnimationFrame).not.toHaveBeenCalled()
  })

  it('recomputes token state immediately after seeking backward', () => {
    const audio = createAudio({ currentTime: 0.75, paused: true })
    const { result } = renderTimeline({ audio, reducedMotion: true })
    const first = document.createElement('span')
    const second = document.createElement('span')

    act(() => {
      result.current.registerToken(
        '0:first',
        {
          lineIndex: 0,
          window: { start: 0, end: 500 },
        },
        first,
      )
      result.current.registerToken(
        '0:second',
        {
          lineIndex: 0,
          window: { start: 500, end: 1000 },
        },
        second,
      )
    })

    expect(first.dataset.lyricsState).toBe('completed')
    expect(second.dataset.lyricsState).toBe('active')

    act(() => {
      audio.currentTime = 0.1
      audio.dispatchEvent(new Event('seeking'))
    })

    expect(first.dataset.lyricsState).toBe('active')
    expect(second.dataset.lyricsState).toBe('future')
  })

  it('tracks overlapping line intervals as an active set', () => {
    const audio = createAudio({ currentTime: 2.5, paused: true })
    const overlappingLines = [
      { start: 1000, end: 4000, tokens: [] },
      { start: 2000, end: 3000, tokens: [] },
      { start: 5000, end: 6000, tokens: [] },
    ]
    const { result } = renderTimeline({ audio, sourceLines: overlappingLines })

    expect(result.current.activeIndexes).toEqual([0, 1])
    expect(result.current.primaryIndex).toBe(1)
  })

  it('publishes the most recently started line when display order differs', () => {
    const audio = createAudio({ currentTime: 2.5, paused: true })
    const displayOrderLines = [
      { start: 2000, end: 4000, tokens: [] },
      { start: 1000, end: 4000, tokens: [] },
    ]
    const { result } = renderTimeline({ audio, sourceLines: displayOrderLines })

    expect(result.current.activeIndexes).toEqual([0, 1])
    expect(result.current.primaryIndex).toBe(0)
  })

  it('keeps a completed word fully revealed', () => {
    const audio = createAudio({ currentTime: 0.25, paused: true })
    const { result } = renderTimeline({ audio })
    const tokenNode = registerToken(
      result,
      '0:stable-completion',
      { start: 0, end: 500 },
      'first',
    )

    expect(tokenNode.dataset.lyricsState).toBe('active')
    expect(sweepAnimationOf(tokenNode).currentTime).toBe(370)

    syncNow(result, 900)

    expect(tokenNode.dataset.lyricsState).toBe('completed')
    expect(sweepAnimationOf(tokenNode).currentTime).toBe(1020)
    characterAnimationsOf(tokenNode).forEach((animation) =>
      expect(animation.currentTime).toBe(1020),
    )
  })

  it('keeps the sweep revealed when release becomes past', () => {
    const audio = createAudio({ currentTime: 0.25, paused: true })
    const { result } = renderTimeline({ audio })
    const tokenNode = registerToken(
      result,
      '0:no-release-blink',
      { start: 0, end: 500 },
      'first',
    )

    const releaseEnd = 1000 + KARAOKE_LINE_RELEASE_MS
    syncNow(result, releaseEnd - 1)
    expect(tokenNode.dataset.lyricsState).toBe('release')
    expect(sweepAnimationOf(tokenNode).currentTime).toBe(1250)

    syncNow(result, releaseEnd)

    expect(tokenNode.dataset.lyricsState).toBe('inactive-past')
    expect(sweepAnimationOf(tokenNode).currentTime).toBe(1250)
  })

  it('marks the whole line for release instead of interpolating each token', () => {
    const audio = createAudio({ currentTime: 0.25, paused: true })
    const { result } = renderTimeline({ audio })
    const lineNode = document.createElement('div')
    act(() => result.current.registerLine(0, lineNode))
    const tokenNode = registerToken(
      result,
      '0:eased-release',
      { start: 0, end: 500 },
      'first',
    )

    syncNow(result, 1000 + KARAOKE_LINE_RELEASE_MS / 2)

    expect(lineNode).toHaveAttribute('data-lifecycle', 'release')
    expect(lineNode).toHaveAttribute('data-active', 'false')
    expect(tokenNode.dataset.lyricsState).toBe('release')
    expect(sweepAnimationOf(tokenNode).currentTime).toBe(1250)
  })

  it('plans a capped letter wave and lets it run natively', () => {
    const audio = createAudio({ currentTime: 0, duration: 6, paused: true })
    const longLines = [
      {
        start: 0,
        end: 5000,
        tokens: [{ start: 500, end: 4500, value: 'super' }],
      },
    ]
    const { result } = renderTimeline({ audio, sourceLines: longLines })
    const tokenNode = registerToken(
      result,
      '0:long-word',
      { start: 500, end: 4500 },
      'super',
    )

    const animations = characterAnimationsOf(tokenNode)
    expect(animations.map((animation) => animation.options.delay)).toEqual([
      0, 35, 70, 105, 140,
    ])
    animations.forEach((animation) => {
      expect(animation.options.duration).toBe(KARAOKE_CHARACTER_WAVE_MS)
      expect(animation.options.fill).toBe('both')
      expect(animation.keyframes[0].transform).toBe('translateY(0)')
      expect(animation.keyframes[1]).toMatchObject({
        offset: KARAOKE_CHARACTER_CREST_RATIO,
        transform: `translateY(-${KARAOKE_CHARACTER_LIFT_EM}em)`,
      })
      expect(animation.keyframes[2].transform).toBe(
        `translateY(-${KARAOKE_CHARACTER_SETTLE_EM}em)`,
      )
    })

    const tokenStart = 500 - KARAOKE_HIGHLIGHT_LEAD_MS
    syncNow(result, tokenStart)
    animations.forEach((animation) => expect(animation.currentTime).toBe(0))

    syncNow(result, tokenStart + KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS / 2)
    animations.forEach((animation) =>
      expect(animation.currentTime).toBe(
        KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS / 2,
      ),
    )
    expect(sweepAnimationOf(tokenNode).currentTime).toBe(
      KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS / 2,
    )

    const completeWaveAt =
      tokenStart +
      KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS +
      KARAOKE_CHARACTER_WAVE_MS
    syncNow(result, completeWaveAt)
    expect(sweepAnimationOf(tokenNode).currentTime).toBe(
      KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS + KARAOKE_CHARACTER_WAVE_MS,
    )
  })

  it('keeps letter motion running after a short token stops highlighting', () => {
    const audio = createAudio({ currentTime: 0, duration: 2, paused: true })
    const shortLines = [
      {
        start: 0,
        end: 1200,
        tokens: [{ start: 500, end: 680, value: 'go' }],
      },
    ]
    const { result } = renderTimeline({ audio, sourceLines: shortLines })
    const tokenNode = registerToken(
      result,
      '0:short-word',
      { start: 500, end: 680 },
      'go',
    )

    const animations = characterAnimationsOf(tokenNode)
    const waveSpan = 180 * KARAOKE_CHARACTER_WAVE_SPAN_RATIO
    expect(animations.map((animation) => animation.options.delay)).toEqual([
      0,
      waveSpan,
    ])

    syncNow(result, 680 - KARAOKE_HIGHLIGHT_LEAD_MS)
    expect(tokenNode.dataset.lyricsState).toBe('completed')
    animations.forEach((animation) => expect(animation.currentTime).toBe(180))

    syncNow(
      result,
      500 - KARAOKE_HIGHLIGHT_LEAD_MS + waveSpan + KARAOKE_CHARACTER_WAVE_MS,
    )
    animations.forEach((animation) =>
      expect(animation.currentTime).toBe(waveSpan + KARAOKE_CHARACTER_WAVE_MS),
    )
  })

  it('gives a single character the fixed wave duration', () => {
    const audio = createAudio({ currentTime: 0, duration: 5, paused: true })
    const sourceLines = [
      {
        start: 0,
        end: 5000,
        tokens: [{ start: 120, end: 4120, value: 'I' }],
      },
    ]
    const { result } = renderTimeline({ audio, sourceLines })
    const tokenNode = registerToken(
      result,
      '0:single-character',
      { start: 120, end: 4120 },
      'I',
    )
    const animation = characterAnimationsOf(tokenNode)[0]

    expect(animation.options.delay).toBe(0)
    expect(animation.options.duration).toBe(KARAOKE_CHARACTER_WAVE_MS)

    const tokenStart = 120 - KARAOKE_HIGHLIGHT_LEAD_MS
    syncNow(result, tokenStart)
    expect(animation.currentTime).toBe(0)
    syncNow(result, tokenStart + KARAOKE_CHARACTER_WAVE_MS / 2)
    expect(animation.currentTime).toBe(KARAOKE_CHARACTER_WAVE_MS / 2)
  })

  it('forces an unfinished character wave to its settled release state', () => {
    const audio = createAudio({ currentTime: 0, duration: 2, paused: true })
    const sourceLines = [
      {
        start: 0,
        end: 600,
        tokens: [{ start: 500, end: 550, value: 'go' }],
      },
    ]
    const { result } = renderTimeline({ audio, sourceLines })
    const tokenNode = registerToken(
      result,
      '0:release-wave',
      { start: 500, end: 550 },
      'go',
    )
    const animations = characterAnimationsOf(tokenNode)

    syncNow(result, 500 - KARAOKE_HIGHLIGHT_LEAD_MS)
    expect(animations[1].currentTime).toBe(0)

    syncNow(result, 600)
    expect(tokenNode.dataset.lyricsState).toBe('release')
    animations.forEach((animation) => {
      expect(animation.currentTime).toBe(50 + KARAOKE_CHARACTER_WAVE_MS)
      expect(animation.playState).toBe('paused')
    })
  })

  it('lets native animations run without resyncing inside the drift window', () => {
    let frameCallback = null
    let now = 0
    window.requestAnimationFrame.mockImplementation((callback) => {
      frameCallback = callback
      return 1
    })
    vi.spyOn(performance, 'now').mockImplementation(() => now)

    const audio = createAudio({ currentTime: 0.2, paused: false })
    const { result } = renderTimeline({ audio })
    const tokenNode = createTokenNode()
    act(() => {
      result.current.registerToken(
        '0:monotonic',
        {
          lineIndex: 0,
          window: { start: 0, end: 1000 },
        },
        tokenNode,
      )
    })
    const animation = sweepAnimationOf(tokenNode)
    expect(animation.currentTime).toBe(320)
    expect(animation.playState).toBe('running')

    now = 100
    audio.currentTime = (200 + now) / 1000
    act(() => frameCallback())
    expect(animation.currentTime).toBe(320)

    now = 160
    audio.currentTime = (200 + now) / 1000
    act(() => frameCallback())
    expect(animation.currentTime).toBe(320)

    // A media clock jump is real drift and does get corrected.
    audio.currentTime = 0.6
    act(() => frameCallback())
    expect(animation.currentTime).toBe(720)
  })

  it('applies the audio playback rate to native animations', () => {
    let frameCallback = null
    let now = 0
    window.requestAnimationFrame.mockImplementation((callback) => {
      frameCallback = callback
      return 1
    })
    vi.spyOn(performance, 'now').mockImplementation(() => now)

    const audio = createAudio({
      currentTime: 0.38,
      duration: 2,
      paused: false,
      playbackRate: 2,
    })
    const sourceLines = [
      {
        start: 0,
        end: 1200,
        tokens: [{ start: 500, end: 1000, value: 'go' }],
      },
    ]
    const { result } = renderTimeline({ audio, sourceLines })
    const tokenNode = registerToken(
      result,
      '0:playback-rate-wave',
      { start: 500, end: 1000 },
      'go',
    )
    const animations = characterAnimationsOf(tokenNode)
    animations.forEach((animation) => expect(animation.playbackRate).toBe(2))

    now = KARAOKE_CLOCK_DRIFT_RESET_MS / 4
    audio.currentTime = (380 + now * 2) / 1000
    act(() => frameCallback())
    animations.forEach((animation) => {
      // Becoming active forces one exact sync, then the browser owns the clock.
      expect(animation.currentTime).toBe(70)
      expect(animation.playState).toBe('running')
    })

    now = 100
    audio.currentTime = (380 + now * 2) / 1000
    act(() => frameCallback())
    expect(animations[0].currentTime).toBe(70)
  })

  it('pauses native animations while hidden and resyncs them on recovery', () => {
    let visibility = 'visible'
    vi.spyOn(window.document, 'visibilityState', 'get').mockImplementation(
      () => visibility,
    )
    const audio = createAudio({
      currentTime: 0.38,
      duration: 2,
      paused: false,
    })
    const sourceLines = [
      {
        start: 0,
        end: 2000,
        tokens: [{ start: 500, end: 1500, value: 'go' }],
      },
    ]
    const { result } = renderTimeline({ audio, sourceLines })
    const tokenNode = registerToken(
      result,
      '0:visibility-wave',
      { start: 500, end: 1500 },
      'go',
    )
    const animations = characterAnimationsOf(tokenNode)

    visibility = 'hidden'
    act(() => {
      window.document.dispatchEvent(new Event('visibilitychange'))
    })
    animations.forEach((animation) =>
      expect(animation.playState).toBe('paused'),
    )

    const characterMotionEnd =
      500 +
      Math.min(
        1000 * KARAOKE_CHARACTER_WAVE_SPAN_RATIO,
        KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS,
      ) +
      KARAOKE_CHARACTER_WAVE_MS
    audio.currentTime = (characterMotionEnd - KARAOKE_HIGHLIGHT_LEAD_MS) / 1000
    visibility = 'visible'
    act(() => {
      window.document.dispatchEvent(new Event('visibilitychange'))
    })

    animations.forEach((animation) => {
      expect(animation.currentTime).toBe(characterMotionEnd - 500)
      expect(animation.playState).toBe('running')
    })
  })

  it('resyncs native animations after an explicit backward seek', () => {
    let frameCallback = null
    let now = 0
    window.requestAnimationFrame.mockImplementation((callback) => {
      frameCallback = callback
      return 1
    })
    vi.spyOn(performance, 'now').mockImplementation(() => now)

    const audio = createAudio({ currentTime: 0.8, paused: false })
    const { result } = renderTimeline({ audio })
    const tokenNode = createTokenNode()
    act(() => {
      result.current.registerToken(
        '0:backward-seek',
        {
          lineIndex: 0,
          window: { start: 0, end: 1000 },
        },
        tokenNode,
      )
    })
    const animation = sweepAnimationOf(tokenNode)

    now = 100
    act(() => frameCallback())
    const beforeSeek = animation.currentTime
    expect(beforeSeek).toBe(1020)

    audio.currentTime = 0.1
    act(() => {
      audio.dispatchEvent(new Event('seeked'))
    })
    now = 120
    act(() => frameCallback())

    expect(animation.currentTime).toBeLessThan(beforeSeek)
    expect(animation.currentTime).toBe(220)
  })

  it('starts and stops requestAnimationFrame with playback visibility', () => {
    const audio = createAudio({ currentTime: 0.25, paused: true })
    renderTimeline({ audio })

    expect(window.requestAnimationFrame).not.toHaveBeenCalled()

    act(() => {
      audio.paused = false
      audio.dispatchEvent(new Event('play'))
    })
    expect(window.requestAnimationFrame).toHaveBeenCalledTimes(1)

    act(() => {
      audio.paused = true
      audio.dispatchEvent(new Event('pause'))
    })
    expect(window.cancelAnimationFrame).toHaveBeenCalled()
  })

  it('rebuilds open-line timing when media duration changes', () => {
    const audio = createAudio({ duration: 10, paused: true })
    const { result } = renderTimeline({
      audio,
      sourceLines: [{ start: 7000, tokens: [] }],
    })

    expect(result.current.timeline.windows[0].end).toBe(10000)

    act(() => {
      audio.duration = 20
      audio.dispatchEvent(new Event('durationchange'))
    })

    expect(result.current.timeline.windows[0].end).toBe(15000)
  })
})
