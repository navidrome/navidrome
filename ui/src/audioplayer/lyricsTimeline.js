import { KARAOKE_SCROLL_PRE_ROLL_MS } from './lyricsKaraokeConstants'

export const finiteTime = (value) => {
  if (value == null || value === '') return null
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}

const upperBound = (items, time, getTime) => {
  let low = 0
  let high = items.length
  while (low < high) {
    const middle = (low + high) >>> 1
    if (getTime(items[middle]) <= time) low = middle + 1
    else high = middle
  }
  return low
}

export const sameIndexes = (left, right) =>
  left.length === right.length &&
  left.every((value, index) => value === right[index])

const getPrimaryActiveIndex = (timeline, indexes) =>
  indexes.reduce((primaryIndex, lineIndex) => {
    if (primaryIndex < 0) return lineIndex
    const primary = timeline?.windows?.[primaryIndex]
    const candidate = timeline?.windows?.[lineIndex]
    if ((candidate?.start ?? -Infinity) !== (primary?.start ?? -Infinity)) {
      return (candidate?.start ?? -Infinity) > (primary?.start ?? -Infinity)
        ? lineIndex
        : primaryIndex
    }
    return lineIndex > primaryIndex ? lineIndex : primaryIndex
  }, -1)

const earliestTokenStart = (line) => {
  const tokens = Array.isArray(line?.tokens) ? line.tokens : []
  let earliest = null
  for (const token of tokens) {
    const start = finiteTime(token?.start)
    if (start != null && (earliest == null || start < earliest))
      earliest = start
  }
  return earliest
}

const latestExplicitTokenEnd = (line) => {
  const tokens = Array.isArray(line?.tokens) ? line.tokens : []
  let latest = null
  for (const token of tokens) {
    const end = finiteTime(token?.end)
    if (end != null && (latest == null || end > latest)) latest = end
  }
  return latest
}

export const buildLyricsTimeline = (
  lines,
  { durationMs = null, fallbackLineDurationMs = 8000 } = {},
) => {
  const sourceLines = Array.isArray(lines) ? lines : []
  const starts = sourceLines.map(
    (line) => finiteTime(line?.start) ?? earliestTokenStart(line),
  )
  const nextTimedStarts = new Array(sourceLines.length).fill(null)
  const timedEntries = starts
    .map((start, lineIndex) => ({ start, lineIndex }))
    .filter((entry) => entry.start != null)
    .sort(
      (left, right) =>
        left.start - right.start || left.lineIndex - right.lineIndex,
    )
  let nextDistinctStart = null
  for (let index = timedEntries.length - 1; index >= 0; ) {
    const groupStart = timedEntries[index].start
    let groupIndex = index
    while (groupIndex >= 0 && timedEntries[groupIndex].start === groupStart) {
      nextTimedStarts[timedEntries[groupIndex].lineIndex] = nextDistinctStart
      groupIndex -= 1
    }
    nextDistinctStart = groupStart
    index = groupIndex
  }

  const trackEnd = finiteTime(durationMs)
  const windows = sourceLines.map((line, lineIndex) => {
    const start = starts[lineIndex]
    let end = finiteTime(line?.end) ?? latestExplicitTokenEnd(line)
    if (end == null) end = nextTimedStarts[lineIndex]
    if (end == null && start != null) {
      end = start + fallbackLineDurationMs
      if (trackEnd != null && trackEnd > start) {
        end = Math.min(end, trackEnd)
      }
    }
    const renderable = line?.renderable !== false
    const intervalValid = start != null && end != null && end > start
    const valid = renderable && intervalValid
    return {
      lineIndex,
      start,
      end,
      renderable,
      intervalValid,
      valid,
      preRollStart:
        start == null ? null : Math.max(0, start - KARAOKE_SCROLL_PRE_ROLL_MS),
      nextTimedStart: nextTimedStarts[lineIndex],
    }
  })

  const events = windows
    .filter((window) => window.valid)
    .flatMap((window) => [
      { time: window.start, type: 'start', lineIndex: window.lineIndex },
      { time: window.end, type: 'end', lineIndex: window.lineIndex },
    ])
    .sort((left, right) => {
      if (left.time !== right.time) return left.time - right.time
      if (left.type !== right.type) return left.type === 'end' ? -1 : 1
      return left.lineIndex - right.lineIndex
    })

  const checkpoints = []
  const active = new Set()
  events.forEach((event, index) => {
    if (event.type === 'start') active.add(event.lineIndex)
    else active.delete(event.lineIndex)
    if ((index + 1) % 64 === 0) {
      checkpoints.push({
        eventIndex: index + 1,
        time: event.time,
        active: Array.from(active),
      })
    }
  })

  const scrollOrder = windows
    .filter((window) => window.valid && window.preRollStart != null)
    .sort(
      (left, right) =>
        left.preRollStart - right.preRollStart ||
        left.start - right.start ||
        left.lineIndex - right.lineIndex,
    )

  return { windows, events, checkpoints, scrollOrder }
}

export const getTimelineScrollTarget = (timeline, time) => {
  const current = finiteTime(time) ?? 0
  const order = timeline?.scrollOrder || []
  const index = upperBound(order, current, (item) => item.preRollStart) - 1
  return index >= 0 ? order[index].lineIndex : -1
}

export class LyricTimelineCursor {
  constructor(timeline) {
    this.timeline = timeline
    this.eventIndex = 0
    this.active = new Set()
    this.lastTime = -Infinity
    this.lastIndexes = []
    this.result = {
      indexes: this.lastIndexes,
      changed: false,
      primaryIndex: -1,
    }
  }

  reset(timeline = this.timeline) {
    this.timeline = timeline
    this.eventIndex = 0
    this.active.clear()
    this.lastTime = -Infinity
    this.lastIndexes = []
    this.result.indexes = this.lastIndexes
    this.result.changed = false
    this.result.primaryIndex = -1
  }

  applyEvent(event) {
    if (event.type === 'start') this.active.add(event.lineIndex)
    else this.active.delete(event.lineIndex)
  }

  seek(time) {
    const events = this.timeline?.events || []
    const checkpoints = this.timeline?.checkpoints || []
    const checkpointIndex =
      upperBound(checkpoints, time, (item) => item.time) - 1
    const checkpoint =
      checkpointIndex >= 0 ? checkpoints[checkpointIndex] : null
    this.active = new Set(checkpoint?.active || [])
    this.eventIndex = checkpoint?.eventIndex || 0
    while (
      this.eventIndex < events.length &&
      events[this.eventIndex].time <= time
    ) {
      this.applyEvent(events[this.eventIndex])
      this.eventIndex += 1
    }
  }

  update(time, forceSeek = false) {
    const current = finiteTime(time) ?? 0
    const events = this.timeline?.events || []
    const previousEventIndex = this.eventIndex
    if (forceSeek || current < this.lastTime) {
      this.seek(current)
    } else {
      while (
        this.eventIndex < events.length &&
        events[this.eventIndex].time <= current
      ) {
        this.applyEvent(events[this.eventIndex])
        this.eventIndex += 1
      }
    }
    this.lastTime = current
    if (!forceSeek && previousEventIndex === this.eventIndex) {
      this.result.changed = false
      return this.result
    }
    const indexes = Array.from(this.active).sort((a, b) => a - b)
    const changed = !sameIndexes(indexes, this.lastIndexes)
    this.lastIndexes = indexes
    this.result.indexes = indexes
    this.result.changed = changed
    this.result.primaryIndex = getPrimaryActiveIndex(this.timeline, indexes)
    return this.result
  }
}

export const resolveKaraokeTokenWindows = (line, lineEndFallback = null) => {
  const tokens = Array.isArray(line?.tokens) ? line.tokens : []
  if (tokens.length === 0) return []

  const lineStart = finiteTime(line?.start)
  const lineEnd = finiteTime(line?.end) ?? finiteTime(lineEndFallback)
  const hasLineWindow =
    lineStart != null && lineEnd != null && lineEnd > lineStart
  const tokenCount = tokens.length
  const windows = new Array(tokenCount)

  for (let tokenIndex = 0; tokenIndex < tokenCount; tokenIndex += 1) {
    const token = tokens[tokenIndex]
    const previousToken = tokenIndex > 0 ? tokens[tokenIndex - 1] : null
    const nextToken =
      tokenIndex + 1 < tokenCount ? tokens[tokenIndex + 1] : null
    const estimatedStart = hasLineWindow
      ? lineStart + ((lineEnd - lineStart) * tokenIndex) / tokenCount
      : null
    const estimatedEnd = hasLineWindow
      ? lineStart + ((lineEnd - lineStart) * (tokenIndex + 1)) / tokenCount
      : null
    const previousEnd =
      finiteTime(previousToken?.end) ?? finiteTime(previousToken?.start)

    let start = finiteTime(token?.start)
    if (start == null) start = previousEnd ?? estimatedStart ?? lineStart

    let end = finiteTime(token?.end)
    if (end == null) {
      const nextStart = finiteTime(nextToken?.start)
      const nextEstimatedStart =
        hasLineWindow && tokenIndex + 1 < tokenCount
          ? lineStart + ((lineEnd - lineStart) * (tokenIndex + 1)) / tokenCount
          : null
      end = nextStart ?? nextEstimatedStart ?? estimatedEnd ?? lineEnd
    }

    if (
      tokenCount === 1 &&
      hasLineWindow &&
      (start == null || end == null || end <= start + 1)
    ) {
      start = lineStart
      end = lineEnd
    }
    if (start != null && end != null && end < start) end = start

    const previousWindow = tokenIndex > 0 ? windows[tokenIndex - 1] : null
    const visualStart =
      start != null && previousWindow?.end != null
        ? Math.max(start, previousWindow.end)
        : start
    const visualEnd =
      end != null && visualStart != null && end <= visualStart
        ? visualStart + 1
        : end
    windows[tokenIndex] = {
      start: visualStart,
      end: visualEnd,
      sourceStart: start,
      sourceEnd: end,
    }
  }

  return windows
}

export const tokenProgressAt = (window, time) => {
  const current = finiteTime(time) ?? 0
  if (!window || window.start == null || window.end == null) return 0
  if (window.end <= window.start) return current >= window.start ? 1 : 0
  return Math.max(
    0,
    Math.min(1, (current - window.start) / (window.end - window.start)),
  )
}
