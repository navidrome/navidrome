// Serializes structured lyrics for download in the source's format, falling
// back to plain text for untimed LRC. Source offsets are included; the
// listener's local timing adjustment is deliberately not exported.

import { buildKaraokeLines, hasUsableKaraokeTiming } from './lyrics'
import { buildSegmentsFromLine } from './lyricsSegments'

const pad = (value, length = 2) => String(value).padStart(length, '0')

export const lyricsFileExtension = (format, lyric) => {
  const normalized = String(format || '')
    .trim()
    .toLowerCase()
  if (normalized === 'ttml') return 'ttml'
  if (normalized === 'lrc' || normalized === 'elrc' || normalized === 'srt') {
    return lyric && !hasUsableKaraokeTiming(buildKaraokeLines(lyric))
      ? 'txt'
      : 'lrc'
  }
  return 'txt'
}

const sanitizePart = (value, fallback) => {
  const cleaned = String(value ?? '')
    .replace(/[\\/:*?"<>|]+/g, '')
    .replace(/\s+/g, ' ')
    .trim()
    .replace(/\.+$/, '')
  return cleaned || fallback
}

export const buildLyricsFileName = ({ artist, title, label, format } = {}) =>
  `${sanitizePart(artist, 'Unknown artist')} - ${sanitizePart(
    title,
    'Unknown title',
  )} (${sanitizePart(label, 'Lyrics')}).${lyricsFileExtension(format)}`

const escapeXml = (value) =>
  String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')

// LRC stamps use centiseconds; timestamps clamp at zero because LRC has no
// negative time. Past 99 minutes the hours form keeps the field width valid.
const lrcStamp = (ms, open = '[', close = ']') => {
  const total = Math.max(0, Math.round(Number(ms) || 0))
  const centis = Math.floor(total / 10)
  const seconds = Math.floor(centis / 100)
  const minutes = Math.floor(seconds / 60)
  if (minutes >= 100) {
    const hours = Math.floor(minutes / 60)
    return `${open}${pad(hours)}:${pad(minutes % 60)}:${pad(
      seconds % 60,
    )}.${pad(centis % 100)}${close}`
  }
  return `${open}${pad(minutes)}:${pad(seconds % 60)}.${pad(
    centis % 100,
  )}${close}`
}

const ttmlClock = (ms) => {
  const total = Math.max(0, Math.round(Number(ms) || 0))
  const hours = Math.floor(total / 3_600_000)
  const minutes = Math.floor(total / 60_000) % 60
  const seconds = Math.floor(total / 1_000) % 60
  return `${pad(hours)}:${pad(minutes)}:${pad(seconds)}.${pad(
    total % 1_000,
    3,
  )}`
}

const lanesForLine = (line) => (line.lanes?.length > 0 ? line.lanes : [line])

const serializeLrcLane = (lane, fallbackStart) => {
  const segments = buildSegmentsFromLine(lane)
  const start = lane.start ?? fallbackStart ?? lane.tokens[0]?.start ?? 0
  let lastStart = Math.max(0, start)
  let hasStamp = false
  let prefix = ''
  const body = segments
    .map(({ text, token }) => {
      if (token?.start == null) {
        if (!hasStamp) {
          prefix += text
          return ''
        }
        return text
      }
      lastStart = Math.max(lastStart, token.start)
      hasStamp = true
      const value = `${lrcStamp(lastStart, '<', '>')}${prefix}${text}`
      prefix = ''
      return value
    })
    .join('')
  const lastToken = [...segments]
    .reverse()
    .find(({ token }) => token?.start != null)?.token
  const endStamp =
    lastToken?.end == null
      ? ''
      : lrcStamp(Math.max(lastStart, lastToken.end), '<', '>')
  const content = `${body}${prefix}${endStamp}`
  return {
    start,
    content,
    hasCues: hasStamp,
    value: `${lrcStamp(start)}${content}`,
  }
}

const serializeLrc = (lines) =>
  lines
    .flatMap((line) => {
      const lanes = lanesForLine(line).map((lane) => ({
        ...serializeLrcLane(lane, line.start),
        background: lane.agentRole === 'bg',
      }))
      const backgrounds = lanes.filter(
        (lane) => lane.background && lane.hasCues,
      )
      const rows = lanes.filter((lane) => !lane.background || !lane.hasCues)
      if (backgrounds.length === 0) return rows
      if (rows.length === 0) {
        const start = line.start ?? backgrounds[0].start
        rows.push({ start, value: lrcStamp(start) })
      }
      // Navidrome's [bg:] extension attaches to the preceding timestamped
      // line. Keep it with that parent while sorting other voice rows.
      rows[0].value += backgrounds
        .map((lane) => `\n[bg:${lane.content}]`)
        .join('')
      return rows
    })
    .sort((left, right) => left.start - right.start)
    .map(({ value }) => value)
    .join('\n')

const serializeTtmlLane = (lane) => {
  let lastStart = 0
  return buildSegmentsFromLine(lane)
    .map(({ text, token }) => {
      if (token?.start == null) return escapeXml(text)
      const start = Math.max(lastStart, token.start)
      lastStart = start
      const end =
        token.end == null
          ? ''
          : ` end="${ttmlClock(Math.max(start, token.end))}"`
      return `<span begin="${ttmlClock(start)}"${end}>${escapeXml(text)}</span>`
    })
    .join('')
}

const ttmlTimingAttributes = ({ start, end }) =>
  `${start == null ? '' : ` begin="${ttmlClock(start)}"`}${
    end == null ? '' : ` end="${ttmlClock(Math.max(start ?? 0, end))}"`
  }`

const serializeTtml = (lyric, lines) => {
  const lang = typeof lyric.lang === 'string' ? lyric.lang.trim() : ''
  const lanes = lines.flatMap(lanesForLine)
  const hasCues = lanes.some((lane) =>
    lane.tokens.some((token) => token.start != null),
  )
  // Wire agent IDs can contain characters forbidden in XML IDs. Keep stable
  // references in this document while preserving the agent's role and name.
  const agentIds = [
    ...new Set(lanes.map((lane) => lane.agentId).filter(Boolean)),
  ]
  const agentRefs = new Map(agentIds.map((id, index) => [id, `v${index + 1}`]))
  const attributes = [
    'xmlns="http://www.w3.org/ns/ttml"',
    'xmlns:itunes="http://music.apple.com/lyric-ttml-internal"',
    ...(agentIds.length > 0
      ? ['xmlns:ttm="http://www.w3.org/ns/ttml#metadata"']
      : []),
    ...(hasCues ? ['itunes:timing="Word"'] : []),
    ...(lang && lang !== 'xxx' ? [`xml:lang="${escapeXml(lang)}"`] : []),
  ]
  const agents = agentIds.map((id) => {
    const lane = lanes.find((candidate) => candidate.agentId === id)
    const type = lane.agentRole === 'group' ? 'group' : 'person'
    const name = lane.agentName
      ? `<ttm:name>${escapeXml(lane.agentName)}</ttm:name>`
      : ''
    return `<ttm:agent xml:id="${agentRefs.get(id)}" type="${type}">${name}</ttm:agent>`
  })
  const paragraphs = lines.map((line) => {
    const content = lanesForLine(line)
      .map((lane) => {
        const value = serializeTtmlLane(lane)
        if (
          !lane.agentId &&
          (lane.start == null || lane.start === line.start) &&
          (lane.end == null || lane.end === line.end)
        ) {
          return value
        }
        const agent = lane.agentId
          ? ` ttm:agent="${agentRefs.get(lane.agentId)}"`
          : ''
        const role = lane.agentRole === 'bg' ? ' ttm:role="x-bg"' : ''
        return `<span${ttmlTimingAttributes(lane)}${agent}${role}>${value}</span>`
      })
      .join(' ')
    return `<p${ttmlTimingAttributes(line)}>${content}</p>`
  })

  return [
    '<?xml version="1.0" encoding="UTF-8"?>',
    `<tt ${attributes.join(' ')}>`,
    ...(agents.length > 0
      ? [`<head><metadata>${agents.join('')}</metadata></head>`]
      : []),
    '<body><div>',
    ...paragraphs,
    '</div></body></tt>',
  ].join('\n')
}

export const serializeLyrics = (lyric, format) => {
  if (!lyric) return ''
  // Reuse the panel's source-offset and voice-lane normalization, with no
  // listener offset, so a downloaded line contains what the listener reads.
  const lines = buildKaraokeLines(lyric)
  switch (lyricsFileExtension(format, lyric)) {
    case 'ttml':
      return serializeTtml(lyric, lines)
    case 'lrc':
      return serializeLrc(lines)
    default:
      return lines.map((line) => line.value).join('\n')
  }
}

export const downloadLyrics = (
  lyric,
  { format, artist, title, label } = {},
) => {
  if (!lyric || typeof document === 'undefined') return
  const text = serializeLyrics(lyric, format)
  if (!text) return

  const extension = lyricsFileExtension(format, lyric)
  const type = extension === 'ttml' ? 'application/xml' : 'text/plain'
  const blob = new Blob([text], { type: `${type};charset=utf-8` })
  const objectUrl = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectUrl
  anchor.download = buildLyricsFileName({
    artist,
    title,
    label,
    format: extension,
  })
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  window.setTimeout(() => URL.revokeObjectURL(objectUrl), 0)
}
