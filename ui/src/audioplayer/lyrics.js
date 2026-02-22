const normalizeLanguageTag = (language) =>
  (language || '').toLowerCase().replace('_', '-')

const KARAOKE_SWITCH_EPSILON_MS = 18
const LYRIC_KIND_MAIN = 'main'
const LYRIC_KIND_TRANSLATION = 'translation'
const LYRIC_KIND_PRONUNCIATION = 'pronunciation'

const padTime = (value) => {
  const str = value.toString()
  return str.length === 1 ? `0${str}` : str
}

const toTime = (value) => {
  const numeric = Number(value)
  return Number.isFinite(numeric) ? numeric : null
}

const compareNullableTime = (a, b) => {
  if (a == null && b == null) {
    return 0
  }
  if (a == null) {
    return 1
  }
  if (b == null) {
    return -1
  }
  return a - b
}

const sortTokensByStart = (tokens) =>
  tokens
    .map((token, order) => ({ ...token, order }))
    .sort((a, b) => {
      const byStart = compareNullableTime(a.start, b.start)
      if (byStart !== 0) {
        return byStart
      }
      const byEnd = compareNullableTime(a.end, b.end)
      if (byEnd !== 0) {
        return byEnd
      }
      return a.order - b.order
    })
    .map(({ order, ...token }) => token)

const languageMatch = (candidate, preferred) => {
  if (!candidate || !preferred) {
    return false
  }
  return (
    candidate === preferred ||
    candidate.startsWith(`${preferred}-`) ||
    preferred.startsWith(`${candidate}-`)
  )
}

const hasTimedLines = (lyric) =>
  lyric &&
  lyric.synced &&
  Array.isArray(lyric.line) &&
  lyric.line.some((line) => Number.isFinite(Number(line.start)))

const normalizeToken = (token) => {
  if (!token) {
    return null
  }
  const value = typeof token.value === 'string' ? token.value : ''
  if (!value.trim()) {
    return null
  }
  return {
    start: toTime(token.start),
    end: toTime(token.end),
    value,
  }
}

const normalizeCueLine = (cueLine, fallbackIndex) => {
  const index = Number.isFinite(Number(cueLine?.index))
    ? Number(cueLine.index)
    : fallbackIndex
  const tokens = sortTokensByStart(
    Array.isArray(cueLine?.cue)
      ? cueLine.cue.map(normalizeToken).filter(Boolean)
      : [],
  )

  return {
    index,
    start: toTime(cueLine?.start),
    end: toTime(cueLine?.end),
    value: typeof cueLine?.value === 'string' ? cueLine.value : '',
    role: typeof cueLine?.role === 'string' ? cueLine.role : '',
    tokens,
  }
}

const normalizeLyricKind = (kind) => {
  const normalized = (kind || '').toLowerCase().trim()
  switch (normalized) {
    case LYRIC_KIND_TRANSLATION:
      return LYRIC_KIND_TRANSLATION
    case LYRIC_KIND_PRONUNCIATION:
      return LYRIC_KIND_PRONUNCIATION
    default:
      return LYRIC_KIND_MAIN
  }
}

const pickLyricByLanguage = (lyrics, preferredLanguage) => {
  if (!Array.isArray(lyrics) || lyrics.length === 0) {
    return null
  }

  const preferred = normalizeLanguageTag(preferredLanguage)
  const preferredBase = preferred.split('-')[0]

  return (
    lyrics.find((lyric) =>
      languageMatch(normalizeLanguageTag(lyric.lang), preferred),
    ) ||
    lyrics.find((lyric) =>
      languageMatch(normalizeLanguageTag(lyric.lang), preferredBase),
    ) ||
    lyrics.find((lyric) =>
      languageMatch(normalizeLanguageTag(lyric.lang), 'en'),
    ) ||
    lyrics[0]
  )
}

const lineTimeWindow = (lines, index) => {
  const line = lines[index]
  if (!line) {
    return { start: null, end: null }
  }

  const start = toTime(line.start)
  const end = toTime(line.end) ?? toTime(lines[index + 1]?.start)
  return { start, end }
}

const buildSyntheticWordTokens = (line, token) => {
  const text = typeof line?.value === 'string' ? line.value : ''
  if (!text.trim()) {
    return null
  }

  const chunks = text.match(/\S+\s*/g) || []
  if (chunks.length < 2) {
    return null
  }

  const normalizedLine = text.replace(/\s+/g, ' ').trim().toLowerCase()
  const normalizedTokenValue = (token?.value || '')
    .replace(/\s+/g, ' ')
    .trim()
    .toLowerCase()
  if (!normalizedTokenValue || !normalizedLine) {
    return null
  }

  const compressedLine = normalizedLine.replace(/\s+/g, '')
  const compressedToken = normalizedTokenValue.replace(/\s+/g, '')
  const tokenLooksLikeWholeLine =
    compressedToken === compressedLine ||
    compressedToken.length >= Math.floor(compressedLine.length * 0.8)
  if (!tokenLooksLikeWholeLine) {
    return null
  }

  const tokenStart = toTime(token?.start)
  const tokenEnd = toTime(token?.end)
  const lineStart = toTime(line?.start)
  const lineEnd = toTime(line?.end)

  const baseStart = tokenStart ?? lineStart
  const baseEnd = tokenEnd ?? lineEnd
  if (
    baseStart == null ||
    baseEnd == null ||
    !Number.isFinite(baseStart) ||
    !Number.isFinite(baseEnd) ||
    baseEnd <= baseStart
  ) {
    return null
  }

  const duration = baseEnd - baseStart
  return chunks.map((chunk, idx) => ({
    start: baseStart + (duration * idx) / chunks.length,
    end: baseStart + (duration * (idx + 1)) / chunks.length,
    value: chunk,
    role: typeof token?.role === 'string' ? token.role : '',
  }))
}

export const hasCueTiming = (structuredLyric) =>
  Boolean(
    structuredLyric &&
    Array.isArray(structuredLyric.cueLine) &&
    structuredLyric.cueLine.some(
      (cueLine) =>
        Array.isArray(cueLine?.cue) &&
        cueLine.cue.some((cue) => Number.isFinite(Number(cue?.start))),
    ),
  )

export const hasStructuredLyricContent = (structuredLyric) =>
  Boolean(
    structuredLyric &&
    ((Array.isArray(structuredLyric.line) &&
      structuredLyric.line.some(
        (line) => typeof line?.value === 'string' && line.value.trim() !== '',
      )) ||
      hasCueTiming(structuredLyric)),
  )

export const getPreferredLyricLanguage = () => {
  if (typeof window !== 'undefined' && window.localStorage) {
    const stored = window.localStorage.getItem('locale')
    if (stored) {
      return stored
    }
  }
  if (typeof navigator !== 'undefined' && navigator.language) {
    return navigator.language
  }
  return 'en'
}

export const selectLyricLayers = (structuredLyrics, preferredLanguage) => {
  if (!Array.isArray(structuredLyrics)) {
    return {
      main: null,
      translation: null,
      pronunciation: null,
    }
  }

  const synced = structuredLyrics.filter(hasTimedLines)
  if (synced.length === 0) {
    return {
      main: null,
      translation: null,
      pronunciation: null,
    }
  }

  const grouped = {
    [LYRIC_KIND_MAIN]: [],
    [LYRIC_KIND_TRANSLATION]: [],
    [LYRIC_KIND_PRONUNCIATION]: [],
  }

  for (const lyric of synced) {
    grouped[normalizeLyricKind(lyric?.kind)].push(lyric)
  }

  const mainCandidates = grouped[LYRIC_KIND_MAIN].length
    ? grouped[LYRIC_KIND_MAIN]
    : synced

  return {
    main: pickLyricByLanguage(mainCandidates, preferredLanguage),
    translation: pickLyricByLanguage(
      grouped[LYRIC_KIND_TRANSLATION],
      preferredLanguage,
    ),
    pronunciation: pickLyricByLanguage(
      grouped[LYRIC_KIND_PRONUNCIATION],
      preferredLanguage,
    ),
  }
}

export const pickStructuredLyric = (structuredLyrics, preferredLanguage) =>
  selectLyricLayers(structuredLyrics, preferredLanguage).main

export const structuredLyricToLrc = (structuredLyric) => {
  if (!structuredLyric || !Array.isArray(structuredLyric.line)) {
    return ''
  }

  let lyricText = ''
  for (const line of structuredLyric.line) {
    const start = Number(line.start)
    if (!Number.isFinite(start) || start < 0) {
      continue
    }

    let time = Math.floor(start / 10)
    const ms = time % 100
    time = Math.floor(time / 100)
    const sec = time % 60
    time = Math.floor(time / 60)
    const min = time % 60

    lyricText += `[${padTime(min)}:${padTime(sec)}.${padTime(ms)}] ${line.value || ''}\n`
  }
  return lyricText
}

export const structuredLyricsToLrc = (structuredLyrics, preferredLanguage) => {
  const selected = pickStructuredLyric(structuredLyrics, preferredLanguage)
  if (!selected) {
    return ''
  }
  return structuredLyricToLrc(selected)
}

export const buildKaraokeLines = (structuredLyric) => {
  if (!structuredLyric) {
    return []
  }

  const baseLines = Array.isArray(structuredLyric.line)
    ? structuredLyric.line
    : []
  const rawCueLines = Array.isArray(structuredLyric.cueLine)
    ? structuredLyric.cueLine
    : []

  const lines =
    rawCueLines.length > 0
      ? (() => {
          const normalizedCueLines = rawCueLines.map(
            (cueLine, fallbackIndex) => {
              const normalized = normalizeCueLine(cueLine, fallbackIndex)
              return {
                ...normalized,
                tokens: normalized.tokens.map((token) => ({
                  ...token,
                  role: normalized.role,
                })),
              }
            },
          )

          const byIndex = new Map()
          for (const cl of normalizedCueLines) {
            if (!byIndex.has(cl.index)) {
              byIndex.set(cl.index, [])
            }
            byIndex.get(cl.index).push(cl)
          }

          return Array.from(byIndex.entries()).map(([index, group]) => {
            const first = group[0]
            const baseLine = baseLines[index] || {}
            const tokens = sortTokensByStart(group.flatMap((cl) => cl.tokens))
            const fallbackStart =
              tokens.find((token) => token.start != null)?.start ?? null
            const fallbackEnd =
              [...tokens].reverse().find((token) => token.end != null)?.end ??
              null
            const value =
              first.value ||
              (typeof baseLine.value === 'string' ? baseLine.value : '') ||
              tokens.map((token) => token.value).join('')

            return {
              index,
              start: first.start ?? toTime(baseLine.start) ?? fallbackStart,
              end: first.end ?? toTime(baseLine.end) ?? fallbackEnd,
              value,
              tokens,
            }
          })
        })()
      : baseLines.map((line, index) => ({
          index,
          start: toTime(line.start),
          end: toTime(line.end),
          value: typeof line.value === 'string' ? line.value : '',
          tokens: [],
        }))

  const normalized = lines
    .filter((line) => line.value || line.tokens.length > 0)
    .sort((a, b) => {
      if (a.start == null && b.start == null) {
        return a.index - b.index
      }
      if (a.start == null) {
        return 1
      }
      if (b.start == null) {
        return -1
      }
      if (a.start !== b.start) {
        return a.start - b.start
      }
      return a.index - b.index
    })
    .map((line) => {
      const nextLine = { ...line }
      if (nextLine.tokens.length === 1) {
        const syntheticTokens = buildSyntheticWordTokens(
          nextLine,
          nextLine.tokens[0],
        )
        if (syntheticTokens) {
          nextLine.tokens = syntheticTokens
        }
      }
      return nextLine
    })

  for (let i = 0; i < normalized.length; i += 1) {
    if (normalized[i].end == null) {
      const nextStart = normalized[i + 1]?.start
      if (nextStart != null) {
        normalized[i].end = nextStart
      }
    }
  }

  return normalized
}

export const resolveKaraokeTokenWindow = (
  line,
  tokenIndex,
  lineEndFallback = null,
) => {
  const tokens = Array.isArray(line?.tokens) ? line.tokens : []
  const token = tokens[tokenIndex]
  if (!token) {
    return { start: null, end: null }
  }

  const prevToken = tokenIndex > 0 ? tokens[tokenIndex - 1] : null
  const nextToken =
    tokenIndex + 1 < tokens.length ? tokens[tokenIndex + 1] : null

  const lineStart = toTime(line?.start)
  const lineEnd = toTime(line?.end) ?? toTime(lineEndFallback)
  const tokenCount = tokens.length
  const hasLineWindow =
    lineStart != null &&
    lineEnd != null &&
    Number.isFinite(lineStart) &&
    Number.isFinite(lineEnd) &&
    lineEnd > lineStart
  const estimatedStart =
    hasLineWindow && tokenCount > 0
      ? lineStart + ((lineEnd - lineStart) * tokenIndex) / tokenCount
      : null
  const estimatedEnd =
    hasLineWindow && tokenCount > 0
      ? lineStart + ((lineEnd - lineStart) * (tokenIndex + 1)) / tokenCount
      : null

  let explicitStartCount = 0
  let explicitEndCount = 0
  const uniqueStarts = new Set()
  const uniqueEnds = new Set()

  for (let i = 0; i < tokenCount; i += 1) {
    const explicitStart = toTime(tokens[i]?.start)
    if (explicitStart != null) {
      explicitStartCount += 1
      uniqueStarts.add(explicitStart)
    }

    const explicitEnd = toTime(tokens[i]?.end)
    if (explicitEnd != null) {
      explicitEndCount += 1
      uniqueEnds.add(explicitEnd)
    }
  }

  const collapsedStarts =
    explicitStartCount > 1 && uniqueStarts.size <= Math.max(1, tokenCount / 4)
  const collapsedEnds =
    explicitEndCount > 1 && uniqueEnds.size <= Math.max(1, tokenCount / 4)
  const shouldForceEstimated =
    hasLineWindow && tokenCount > 1 && (collapsedStarts || collapsedEnds)

  if (shouldForceEstimated) {
    return {
      start: estimatedStart,
      end: estimatedEnd,
    }
  }
  const prevEnd = toTime(prevToken?.end) ?? toTime(prevToken?.start)

  let start = toTime(token.start)
  if (start == null) {
    start = prevEnd ?? estimatedStart ?? lineStart
  }

  let end = toTime(token.end)
  if (end == null) {
    const nextDirectStart = toTime(nextToken?.start)
    const nextEstimatedStart =
      hasLineWindow && tokenIndex + 1 < tokenCount
        ? lineStart + ((lineEnd - lineStart) * (tokenIndex + 1)) / tokenCount
        : null
    end = nextDirectStart ?? nextEstimatedStart ?? estimatedEnd ?? lineEnd
  }

  if (
    tokenCount === 1 &&
    hasLineWindow &&
    (start == null || end == null || end <= start + 1)
  ) {
    start = lineStart
    end = lineEnd
  }

  if (start != null && end != null && end < start) {
    end = start
  }

  return { start, end }
}

export const getActiveKaraokeState = (lines, currentTimeMs) => {
  if (!Array.isArray(lines) || lines.length === 0) {
    return { lineIndex: -1, tokenIndex: -1 }
  }

  const current = Number.isFinite(Number(currentTimeMs))
    ? Number(currentTimeMs)
    : 0
  let lineIndex = 0
  for (let i = 0; i < lines.length; i += 1) {
    const lineStart = toTime(lines[i]?.start)
    if (lineStart == null || lineStart <= current + KARAOKE_SWITCH_EPSILON_MS) {
      lineIndex = i
      continue
    }
    break
  }

  for (let i = lineIndex; i >= 0; i -= 1) {
    const lineStart = toTime(lines[i]?.start)
    const lineEnd = toTime(lines[i]?.end) ?? toTime(lines[i + 1]?.start)
    if (lineStart != null && current + KARAOKE_SWITCH_EPSILON_MS < lineStart) {
      continue
    }
    if (lineEnd == null || current <= lineEnd + KARAOKE_SWITCH_EPSILON_MS) {
      lineIndex = i
      break
    }
  }

  const activeLine = lines[lineIndex] || null
  const tokens = Array.isArray(activeLine?.tokens) ? activeLine.tokens : []
  let tokenIndex = -1
  for (let i = 0; i < tokens.length; i += 1) {
    const { start: tokenStart, end: tokenEnd } = resolveKaraokeTokenWindow(
      activeLine,
      i,
      lines[lineIndex + 1]?.start,
    )
    if (
      tokenStart == null ||
      tokenStart <= current + KARAOKE_SWITCH_EPSILON_MS
    ) {
      tokenIndex = i
      if (tokenEnd != null && current <= tokenEnd + KARAOKE_SWITCH_EPSILON_MS) {
        break
      }
      continue
    }
    break
  }

  return { lineIndex, tokenIndex }
}

export const findLayerLineIndexForMain = (mainLines, layerLines, mainIndex) => {
  if (
    !Array.isArray(mainLines) ||
    !Array.isArray(layerLines) ||
    mainLines.length === 0 ||
    layerLines.length === 0 ||
    mainIndex < 0 ||
    mainIndex >= mainLines.length
  ) {
    return -1
  }

  const { start: mainStart, end: mainEnd } = lineTimeWindow(
    mainLines,
    mainIndex,
  )

  if (mainStart == null) {
    return -1
  }
  const mainWindowEnd = mainEnd ?? mainStart
  const mainWindowDuration = Math.max(0, mainWindowEnd - mainStart)
  const maxDelta = Math.max(550, Math.min(1400, mainWindowDuration + 420))

  let bestIdx = -1
  let bestScore = Number.POSITIVE_INFINITY

  for (let i = 0; i < layerLines.length; i += 1) {
    const { start, end } = lineTimeWindow(layerLines, i)

    if (start != null && end != null) {
      const overlap = Math.min(end, mainEnd ?? end) - Math.max(start, mainStart)
      if (overlap >= 0) {
        const score = Math.abs(start - mainStart) + Math.abs(i - mainIndex) * 30
        if (score < bestScore) {
          bestScore = score
          bestIdx = i
        }
        continue
      }
    }

    if (start != null) {
      if (Math.abs(start - mainStart) > maxDelta) {
        continue
      }
      const score = Math.abs(start - mainStart) + Math.abs(i - mainIndex) * 45
      if (score < bestScore) {
        bestScore = score
        bestIdx = i
      }
    }
  }

  return bestIdx
}

export const resolveLayerLineForMain = (mainLines, layerLines, mainIndex) => {
  const index = findLayerLineIndexForMain(mainLines, layerLines, mainIndex)
  return {
    index,
    line: index >= 0 ? layerLines[index] : null,
  }
}
