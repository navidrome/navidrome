// Serializes a structured lyric (already parsed by Navidrome) back into the
// format its source reported, for download-to-device. Timestamps are absolute:
// the source's own offset is already folded into them by the response, and the
// listener's local nudge is deliberately not exported.

const pad = (value, length = 2) => String(value).padStart(length, '0')

export const lyricsFileExtension = (format) => {
  const normalized = String(format || '')
    .trim()
    .toLowerCase()
  if (normalized === 'ttml') return 'ttml'
  if (normalized === 'lrc' || normalized === 'elrc' || normalized === 'srt') {
    return 'lrc'
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

const cuesByLineIndex = (lyric) => {
  const byIndex = new Map()
  for (const cueLine of Array.isArray(lyric.cueLine) ? lyric.cueLine : []) {
    const index = Number(cueLine?.index)
    if (!Number.isInteger(index)) continue
    if (!byIndex.has(index)) byIndex.set(index, [])
    byIndex.get(index).push(cueLine)
  }
  return byIndex
}

const timedCues = (groups) =>
  groups.flatMap((cueLine) =>
    (Array.isArray(cueLine.cue) ? cueLine.cue : []).filter(
      (cue) => cue?.start != null,
    ),
  )

const serializeLrc = (lyric, lines, baseOffset) => {
  const cuesByIndex = cuesByLineIndex(lyric)
  const withBase = (value) =>
    value == null ? value : Number(value) + baseOffset
  return lines
    .map((line, index) => {
      const cues = timedCues(cuesByIndex.get(index) || [])
      if (cues.length === 0) {
        return `${lrcStamp(withBase(line?.start))}${line?.value ?? ''}`
      }
      const body = cues
        .map(
          (cue) =>
            `${lrcStamp(withBase(cue.start), '<', '>')}${cue.value ?? ''}`,
        )
        .join('')
      return `${lrcStamp(withBase(line?.start))}${body}`
    })
    .join('\n')
}

const serializeTtml = (lyric, lines, baseOffset) => {
  const lang = typeof lyric.lang === 'string' ? lyric.lang.trim() : ''
  const cuesByIndex = cuesByLineIndex(lyric)
  const withBase = (value) =>
    value == null ? value : Number(value) + baseOffset
  const hasCues = [...cuesByIndex.values()].some(
    (groups) => timedCues(groups).length > 0,
  )
  const attributes = [
    'xmlns="http://www.w3.org/ns/ttml"',
    'xmlns:itunes="http://music.apple.com/lyric-ttml-internal"',
    ...(hasCues ? ['itunes:timing="Word"'] : []),
    ...(lang && lang !== 'xxx' ? [`xml:lang="${escapeXml(lang)}"`] : []),
  ]
  const paragraphs = lines.map((line, index) => {
    const begin =
      line?.start == null ? '' : ` begin="${ttmlClock(withBase(line.start))}"`
    const end =
      line?.end == null ? '' : ` end="${ttmlClock(withBase(line.end))}"`
    const cues = timedCues(cuesByIndex.get(index) || [])
    const content =
      cues.length > 0
        ? cues
            .map(
              (cue) =>
                `<span begin="${ttmlClock(withBase(cue.start))}"${
                  cue.end == null
                    ? ''
                    : ` end="${ttmlClock(withBase(cue.end))}"`
                }>${escapeXml(cue.value)}</span>`,
            )
            .join('')
        : escapeXml(line?.value)
    return `<p${begin}${end}>${content}</p>`
  })

  return [
    '<?xml version="1.0" encoding="UTF-8"?>',
    `<tt ${attributes.join(' ')}>`,
    '<body><div>',
    ...paragraphs,
    '</div></body></tt>',
  ].join('\n')
}

export const serializeLyrics = (lyric, format) => {
  if (!lyric) return ''
  const lines = Array.isArray(lyric.line) ? lyric.line : []
  // The source's own offset is a separate field on the response; fold it in so
  // the exported timing matches what the panel plays (the listener's local
  // nudge intentionally stays out).
  const baseOffset = Number(lyric.offset) || 0
  switch (lyricsFileExtension(format)) {
    case 'ttml':
      return serializeTtml(lyric, lines, baseOffset)
    case 'lrc':
      return serializeLrc(lyric, lines, baseOffset)
    default:
      return lines.map((line) => line?.value ?? '').join('\n')
  }
}

export const downloadLyrics = (
  lyric,
  { format, artist, title, label } = {},
) => {
  if (!lyric || typeof document === 'undefined') return
  const text = serializeLyrics(lyric, format)
  if (!text) return

  const extension = lyricsFileExtension(format)
  const type = extension === 'ttml' ? 'application/xml' : 'text/plain'
  const blob = new Blob([text], { type: `${type};charset=utf-8` })
  const objectUrl = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectUrl
  anchor.download = buildLyricsFileName({ artist, title, label, format })
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  window.setTimeout(() => URL.revokeObjectURL(objectUrl), 0)
}
