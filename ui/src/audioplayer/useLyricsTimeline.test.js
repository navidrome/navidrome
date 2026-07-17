import { act } from '@testing-library/react'
import { renderHook } from '@testing-library/react-hooks'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { KARAOKE_CHARACTER_LIFT_PX } from './lyricsKaraokeConstants'
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

const presentation = {
  rgb: [255, 255, 255],
  futureAlpha: 0.34,
  activeAlpha: 1,
  futureColor: 'rgba(255, 255, 255, 0.34)',
  doneColor: 'rgba(255, 255, 255, 1)',
  gradient: 'linear-gradient(90deg, white, transparent)',
  useCrossfade: false,
}

const createTokenNode = (text = '') => {
  const tokenNode = document.createElement('span')
  Array.from(text).forEach((character) => {
    const node = document.createElement('span')
    node.dataset.lyricsCharacter = 'true'
    node.textContent = character
    tokenNode.appendChild(node)
  })
  return tokenNode
}

const registerToken = (result, key, window, text = '') => {
  const tokenNode = createTokenNode(text)
  act(() => {
    result.current.registerToken(
      key,
      { lineIndex: 0, window, presentation },
      tokenNode,
    )
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
          presentation,
        },
        tokenNode,
      )
    })

    expect(result.current.activeIndexes).toEqual([0])
    expect(lineNode.dataset.active).toBe('true')
    expect(tokenNode.dataset.lyricsState).toBe('active')
    expect(
      Number(tokenNode.style.getPropertyValue('--lyrics-progress')),
    ).toBeCloseTo(0.74, 2)
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
          presentation,
        },
        first,
      )
      result.current.registerToken(
        '0:second',
        {
          lineIndex: 0,
          window: { start: 500, end: 1000 },
          presentation,
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

  it('keeps the same gradient paint when an active word completes', () => {
    const audio = createAudio({ currentTime: 0.25, paused: true })
    const { result } = renderTimeline({ audio })
    const tokenNode = registerToken(
      result,
      '0:stable-completion',
      { start: 0, end: 500 },
      'first',
    )

    const activeBackground = tokenNode.style.backgroundImage
    expect(tokenNode.dataset.lyricsState).toBe('active')
    expect(tokenNode.style.color).toBe('transparent')

    syncNow(result, 600)

    expect(tokenNode.dataset.lyricsState).toBe('completed')
    expect(tokenNode.style.backgroundImage).toBe(activeBackground)
    expect(tokenNode.style.color).toBe('transparent')
    expect(tokenNode.style.webkitTextFillColor).toBe('transparent')
    tokenNode
      .querySelectorAll('[data-lyrics-character="true"]')
      .forEach((character) =>
        expect(character.style.transform).toBe('translateY(-1.5000px)'),
      )
  })

  it('keeps gradient paint and opacity continuous when release becomes past', () => {
    const audio = createAudio({ currentTime: 0.25, paused: true })
    const { result } = renderTimeline({ audio })
    const tokenNode = registerToken(
      result,
      '0:no-release-blink',
      { start: 0, end: 500 },
      'first',
    )

    syncNow(result, 1219)
    const gradient = tokenNode.style.backgroundImage
    const releaseAlpha = Number(
      tokenNode.style.getPropertyValue('--lyrics-token-active-alpha'),
    )
    expect(tokenNode.dataset.lyricsState).toBe('release')
    expect(tokenNode.style.color).toBe('transparent')
    expect(tokenNode.style.opacity).toBe('1')

    syncNow(result, 1220)

    const pastAlpha = Number(
      tokenNode.style.getPropertyValue('--lyrics-token-active-alpha'),
    )
    expect(tokenNode.dataset.lyricsState).toBe('inactive-past')
    expect(tokenNode.style.backgroundImage).toBe(gradient)
    expect(tokenNode.style.color).toBe('transparent')
    expect(tokenNode.style.webkitTextFillColor).toBe('transparent')
    expect(tokenNode.style.opacity).toBe('1')
    expect(Math.abs(pastAlpha - releaseAlpha)).toBeLessThan(0.01)
    expect(pastAlpha).toBeCloseTo(presentation.futureAlpha, 5)
    expect(tokenNode.style.getPropertyValue('--lyrics-progress')).toBe('1')
  })

  it('uses the complete token duration for the character phase wave', () => {
    const audio = createAudio({ currentTime: 0, duration: 5, paused: true })
    const longLines = [
      {
        start: 0,
        end: 4000,
        tokens: [{ start: 0, end: 4000, value: 'super' }],
      },
    ]
    const { result } = renderTimeline({ audio, sourceLines: longLines })
    const tokenNode = registerToken(
      result,
      '0:long-word',
      { start: 0, end: 4000 },
      'super',
    )

    const characters = Array.from(
      tokenNode.querySelectorAll('[data-lyrics-character="true"]'),
    )
    expect(characters[0].style.backgroundImage).toBe(
      tokenNode.style.backgroundImage,
    )
    characters.forEach((character) =>
      expect(character.style.transform).not.toMatch(/^translateY\(\d/),
    )
    const firstTransforms = []
    ;[0, 160, 320, 480].forEach((time) => {
      syncNow(result, time)
      firstTransforms.push(characters[0].style.transform)
    })

    expect(new Set(firstTransforms).size).toBe(firstTransforms.length)
    firstTransforms.forEach((transform) =>
      expect(transform).toMatch(/^translateY\(-\d+\.\d{4}px\)$/),
    )

    syncNow(result, 1320)
    const offsets = characters.map((character) =>
      Math.abs(
        Number.parseFloat(
          character.style.transform.match(/-?\d+\.\d+/)?.[0] || '0',
        ),
      ),
    )
    offsets.forEach((offset) => expect(offset).toBeGreaterThanOrEqual(0))
    expect(offsets.filter((offset) => offset > 0)).toHaveLength(4)
    expect(offsets[0]).toBeGreaterThan(offsets[1])
    expect(offsets[1]).toBeGreaterThan(offsets[2])
    expect(offsets[2]).toBeGreaterThan(offsets[3])
    expect(offsets[4]).toBe(0)

    syncNow(result, 3600)
    expect(characters[4].style.transform).not.toBe(
      `translateY(-${KARAOKE_CHARACTER_LIFT_PX.toFixed(4)}px)`,
    )

    syncNow(result, 3880)
    characters.forEach((character) =>
      expect(character.style.transform).toBe(
        `translateY(-${KARAOKE_CHARACTER_LIFT_PX.toFixed(4)}px)`,
      ),
    )
  })

  it('keeps the normalized wave shape for short token durations', () => {
    const audio = createAudio({ currentTime: 0, duration: 1, paused: true })
    const shortLines = [
      {
        start: 0,
        end: 180,
        tokens: [{ start: 0, end: 180, value: 'go' }],
      },
    ]
    const { result } = renderTimeline({ audio, sourceLines: shortLines })
    const tokenNode = registerToken(
      result,
      '0:short-word',
      { start: 0, end: 180 },
      'go',
    )
    syncNow(result, 0)

    const characters = tokenNode.querySelectorAll(
      '[data-lyrics-character="true"]',
    )
    expect(characters[0].style.transform).not.toBe('')
    expect(characters[1].style.transform).not.toBe(
      `translateY(-${KARAOKE_CHARACTER_LIFT_PX.toFixed(4)}px)`,
    )

    syncNow(result, 60)
    expect(characters[1].style.transform).toBe(
      `translateY(-${KARAOKE_CHARACTER_LIFT_PX.toFixed(4)}px)`,
    )
  })

  it('keeps interpolated playback time monotonic between coarse media updates', () => {
    let frameCallback = null
    let now = 0
    window.requestAnimationFrame.mockImplementation((callback) => {
      frameCallback = callback
      return 1
    })
    vi.spyOn(performance, 'now').mockImplementation(() => now)

    const audio = createAudio({ currentTime: 0.2, paused: false })
    const { result } = renderTimeline({ audio })
    const tokenNode = document.createElement('span')
    act(() => {
      result.current.registerToken(
        '0:monotonic',
        {
          lineIndex: 0,
          window: { start: 0, end: 1000 },
          presentation,
        },
        tokenNode,
      )
    })

    now = 100
    act(() => frameCallback())
    const firstProgress = Number(
      tokenNode.style.getPropertyValue('--lyrics-progress'),
    )
    now = 180
    act(() => frameCallback())
    const secondProgress = Number(
      tokenNode.style.getPropertyValue('--lyrics-progress'),
    )

    expect(secondProgress).toBeGreaterThanOrEqual(firstProgress)
  })

  it('resets monotonic interpolation after an explicit backward seek', () => {
    let frameCallback = null
    let now = 0
    window.requestAnimationFrame.mockImplementation((callback) => {
      frameCallback = callback
      return 1
    })
    vi.spyOn(performance, 'now').mockImplementation(() => now)

    const audio = createAudio({ currentTime: 0.8, paused: false })
    const { result } = renderTimeline({ audio })
    const tokenNode = registerToken(result, '0:backward-seek', {
      start: 0,
      end: 1000,
    })

    now = 100
    act(() => frameCallback())
    const beforeSeek = Number(
      tokenNode.style.getPropertyValue('--lyrics-progress'),
    )

    audio.currentTime = 0.1
    act(() => {
      audio.dispatchEvent(new Event('seeked'))
    })
    now = 120
    act(() => frameCallback())

    const afterSeek = Number(
      tokenNode.style.getPropertyValue('--lyrics-progress'),
    )
    expect(afterSeek).toBeLessThan(beforeSeek)
    expect(afterSeek).toBeCloseTo(0.24, 2)
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
