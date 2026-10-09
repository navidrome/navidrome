import { describe, expect, it } from 'vitest'
import {
  buildLyricsTimeline,
  getTimelineScrollTarget,
  LyricTimelineCursor,
  resolveKaraokeTokenWindows,
  tokenProgressAt,
} from './lyricsTimeline'
import { KARAOKE_SCROLL_PRE_ROLL_MS } from './lyricsKaraokeConstants'

describe('lyricsTimeline', () => {
  it('supports overlapping active lines and backward seeks', () => {
    const timeline = buildLyricsTimeline([
      { start: 1000, end: 3000, tokens: [] },
      { start: 2000, end: 4000, tokens: [] },
    ])
    const cursor = new LyricTimelineCursor(timeline)

    expect(cursor.update(1500, true).indexes).toEqual([0])
    expect(cursor.update(2500).indexes).toEqual([0, 1])
    expect(cursor.update(3500).indexes).toEqual([1])
    expect(cursor.update(1500).indexes).toEqual([0])
  })

  it('excludes zero and negative duration intervals', () => {
    const timeline = buildLyricsTimeline([
      { start: 1000, end: 1000, tokens: [] },
      { start: 2000, end: 1500, tokens: [] },
      { start: 3000, end: 4000, tokens: [] },
    ])

    expect(timeline.events).toEqual([
      { time: 3000, type: 'start', lineIndex: 2 },
      { time: 4000, type: 'end', lineIndex: 2 },
    ])
  })

  it('uses the next timed line across untimed display rows', () => {
    const timeline = buildLyricsTimeline([
      { start: 1000, tokens: [] },
      { value: 'untimed annotation', tokens: [] },
      { start: 5000, end: 6000, tokens: [] },
    ])

    expect(timeline.windows[0].end).toBe(5000)
    expect(timeline.windows[0].nextTimedStart).toBe(5000)
    expect(timeline.windows[1].valid).toBe(false)
  })

  it('uses chronological timing independently from display order', () => {
    const timeline = buildLyricsTimeline([
      { start: 3000, end: 5000, value: 'Displayed first', tokens: [] },
      { start: 1000, end: 4000, value: 'Displayed second', tokens: [] },
      { start: 2000, value: 'Displayed third', tokens: [] },
    ])
    const cursor = new LyricTimelineCursor(timeline)

    expect(timeline.windows[2].end).toBe(3000)
    expect(cursor.update(2500, true)).toMatchObject({
      indexes: [1, 2],
      primaryIndex: 2,
    })
    expect(cursor.update(3500)).toMatchObject({
      indexes: [0, 1],
      primaryIndex: 0,
    })
  })

  it('uses timed blank markers as boundaries without activating or scrolling to them', () => {
    const timeline = buildLyricsTimeline([
      {
        start: 1000,
        value: 'Before pause',
        tokens: [],
        renderable: true,
      },
      { start: 2000, value: '', tokens: [], renderable: false },
      {
        start: 4000,
        end: 5000,
        value: 'After pause',
        tokens: [],
        renderable: true,
      },
    ])
    const cursor = new LyricTimelineCursor(timeline)

    expect(timeline.windows[0].end).toBe(2000)
    expect(timeline.windows[1]).toMatchObject({
      renderable: false,
      intervalValid: true,
      valid: false,
    })
    expect(cursor.update(2500, true).indexes).toEqual([])
    expect(timeline.scrollOrder.map((window) => window.lineIndex)).toEqual([
      0, 2,
    ])
  })

  it('bounds a final open line when media duration is unavailable', () => {
    const timeline = buildLyricsTimeline([{ start: 7000, tokens: [] }], {
      fallbackLineDurationMs: 8000,
    })

    expect(timeline.windows[0]).toMatchObject({
      start: 7000,
      end: 15000,
      valid: true,
    })
  })

  it('keeps a start-only line active until the next timestamp', () => {
    const timeline = buildLyricsTimeline([
      { start: 1000, value: 'A lyric line', tokens: [] },
      { start: 5000, end: 6000, tokens: [] },
    ])
    const cursor = new LyricTimelineCursor(timeline)

    expect(timeline.windows[0].end).toBe(5000)
    expect(cursor.update(4999, true).indexes).toEqual([0])
    expect(cursor.update(5000).indexes).toEqual([1])
  })

  it('caps a final open line with track duration', () => {
    const timeline = buildLyricsTimeline([{ start: 7000, tokens: [] }], {
      durationMs: 10000,
    })

    expect(timeline.windows[0]).toMatchObject({
      start: 7000,
      end: 10000,
      valid: true,
    })
  })

  it('ignores a stale track duration that precedes the final line', () => {
    const timeline = buildLyricsTimeline([{ start: 7000, tokens: [] }], {
      durationMs: 5000,
      fallbackLineDurationMs: 8000,
    })

    expect(timeline.windows[0]).toMatchObject({
      start: 7000,
      end: 15000,
      valid: true,
    })
  })

  it('seeks through checkpoints in timelines longer than 32 lines', () => {
    const timeline = buildLyricsTimeline(
      Array.from({ length: 40 }, (_, index) => ({
        start: index * 1000,
        end: (index + 1) * 1000,
        tokens: [],
      })),
    )
    const cursor = new LyricTimelineCursor(timeline)

    expect(timeline.checkpoints.length).toBeGreaterThan(0)
    expect(cursor.update(35500, true)).toMatchObject({
      indexes: [35],
      primaryIndex: 35,
    })
  })

  it('moves the scroll target at pre-roll without changing active state', () => {
    const timeline = buildLyricsTimeline([
      { start: 1000, end: 1500, tokens: [] },
      { start: 3000, end: 3500, tokens: [] },
    ])

    expect(getTimelineScrollTarget(timeline, 0)).toBe(-1)
    expect(
      getTimelineScrollTarget(timeline, 3000 - KARAOKE_SCROLL_PRE_ROLL_MS),
    ).toBe(1)
  })

  it('precomputes token windows once and keeps progress deterministic', () => {
    const line = {
      start: 1000,
      end: 3000,
      tokens: [
        { start: 1000, value: 'one' },
        { start: 2000, end: 3000, value: 'two' },
      ],
    }
    const windows = resolveKaraokeTokenWindows(line)

    expect(windows).toEqual([
      { start: 1000, end: 2000, sourceStart: 1000, sourceEnd: 2000 },
      { start: 2000, end: 3000, sourceStart: 2000, sourceEnd: 3000 },
    ])
    expect(tokenProgressAt(windows[0], 1500)).toBe(0.5)
    expect(tokenProgressAt(windows[0], 2500)).toBe(1)
  })
})
