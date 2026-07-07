const normalizeLanguageTag = (language) =>
  (language || '').toLowerCase().replace(/_/g, '-')

const LYRIC_KIND_MAIN = 'main'
const LYRIC_KIND_TRANSLATION = 'translation'
const LYRIC_KIND_PRONUNCIATION = 'pronunciation'

const toTime = (value) => {
  if (value == null || value === '') {
    return null
  }
  const numeric = Number(value)
  return Number.isFinite(numeric) ? numeric : null
}

const applyTimeOffset = (value, offset = 0) => {
  const time = toTime(value)
  return time == null ? null : time + offset
}

const toByteOffset = (value) => {
  if (value == null || value === '') {
    return null
  }
  const numeric = Number(value)
  if (!Number.isInteger(numeric) || numeric < 0) {
    return null
  }
  return numeric
}

const compareNullableTime = (a, b) => {
  if (a == null && b == null) return 0
  if (a == null) return 1
  if (b == null) return -1
  return a - b
}

const sortTokensByStart = (tokens) =>
  tokens
    .map((token, order) => ({ ...token, order }))
    .sort((a, b) => {
      const byStart = compareNullableTime(a.start, b.start)
      if (byStart !== 0) return byStart
      const byEnd = compareNullableTime(a.end, b.end)
      if (byEnd !== 0) return byEnd
      return a.order - b.order
    })
    .map(({ order, ...token }) => token)

const languageMatch = (candidate, preferred) => {
  if (!candidate || !preferred) return false
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

const preferTimedLyrics = (lyrics) => {
  const timed = lyrics.filter(hasTimedLines)
  return timed.length > 0 ? timed : lyrics
}

const normalizeToken = (token, offset = 0) => {
  if (!token) return null
  const value = typeof token.value === 'string' ? token.value : ''
  if (value.length === 0) return null
  const byteStart = toByteOffset(token.byteStart)
  const byteEnd = toByteOffset(token.byteEnd)
  return {
    start: applyTimeOffset(token.start, offset),
    end: applyTimeOffset(token.end, offset),
    value,
    ...(byteStart != null ? { byteStart } : {}),
    ...(byteEnd != null ? { byteEnd } : {}),
  }
}

const utf8BytesForCodePoint = (codePoint) => {
  if (codePoint <= 0x7f) return 1
  if (codePoint <= 0x7ff) return 2
  if (codePoint <= 0xffff) return 3
  return 4
}

export const utf8ByteOffsetToCodeUnitIndex = (text, targetByteOffset) => {
  if (typeof text !== 'string' || text.length === 0) return 0

  const target = toByteOffset(targetByteOffset)
  if (target == null || target <= 0) return 0

  let byteOffset = 0
  let index = 0
  while (index < text.length) {
    if (byteOffset >= target) return index
    const codePoint = text.codePointAt(index)
    byteOffset += utf8BytesForCodePoint(codePoint)
    index += codePoint > 0xffff ? 2 : 1
  }

  return text.length
}

export const utf8ByteRangeToCodeUnitRange = (text, byteStart, byteEnd) => {
  if (typeof text !== 'string') return null

  const start = toByteOffset(byteStart)
  const end = toByteOffset(byteEnd)
  if (start == null || end == null || end < start) return null

  const startIndex = utf8ByteOffsetToCodeUnitIndex(text, start)
  const endIndex = utf8ByteOffsetToCodeUnitIndex(text, end + 1)
  if (
    startIndex >= endIndex ||
    startIndex > text.length ||
    endIndex > text.length
  ) {
    return null
  }

  return {
    start: startIndex,
    end: endIndex,
    text: text.slice(startIndex, endIndex),
  }
}

const buildAgentLookup = (structuredLyric) => {
  const lookup = new Map()
  const agents = Array.isArray(structuredLyric?.agents)
    ? structuredLyric.agents
    : []
  for (const agent of agents) {
    const id = typeof agent?.id === 'string' ? agent.id : ''
    if (!id || lookup.has(id)) continue
    lookup.set(id, {
      id,
      role: typeof agent?.role === 'string' ? agent.role : '',
      name: typeof agent?.name === 'string' ? agent.name : '',
    })
  }
  return lookup
}

const deriveUiRole = (agent) => {
  if (!agent?.role || agent.role === 'main') return ''
  return agent.role
}

const normalizeCueLine = (cueLine, fallbackIndex, agentLookup, offset = 0) => {
  const index = Number.isFinite(Number(cueLine?.index))
    ? Number(cueLine.index)
    : fallbackIndex
  const agentId = typeof cueLine?.agentId === 'string' ? cueLine.agentId : ''
  const agent = agentId ? agentLookup.get(agentId) || null : null
  const fallbackRole = typeof cueLine?.role === 'string' ? cueLine.role : ''
  const tokens = sortTokensByStart(
    Array.isArray(cueLine?.cue)
      ? cueLine.cue.map((cue) => normalizeToken(cue, offset)).filter(Boolean)
      : [],
  )

  return {
    index,
    start: applyTimeOffset(cueLine?.start, offset),
    end: applyTimeOffset(cueLine?.end, offset),
    value: typeof cueLine?.value === 'string' ? cueLine.value : '',
    role: agent ? deriveUiRole(agent) : fallbackRole,
    agentId,
    agentRole: agent?.role || fallbackRole,
    agentName: agent?.name || '',
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
  if (!Array.isArray(lyrics) || lyrics.length === 0) return null

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
  if (!line) return { start: null, end: null }

  const start = toTime(line.start)
  const end = toTime(line.end) ?? toTime(lines[index + 1]?.start)
  return { start, end }
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
    if (stored) return stored
  }
  if (typeof navigator !== 'undefined' && navigator.language) {
    return navigator.language
  }
  return 'en'
}

export const selectLyricLayers = (structuredLyrics, preferredLanguage) => {
  if (!Array.isArray(structuredLyrics)) {
    return { main: null, translation: null, pronunciation: null }
  }

  const available = structuredLyrics.filter(hasStructuredLyricContent)
  if (available.length === 0) {
    return { main: null, translation: null, pronunciation: null }
  }

  const grouped = {
    [LYRIC_KIND_MAIN]: [],
    [LYRIC_KIND_TRANSLATION]: [],
    [LYRIC_KIND_PRONUNCIATION]: [],
  }

  for (const lyric of available) {
    grouped[normalizeLyricKind(lyric?.kind)].push(lyric)
  }

  const mainCandidates = grouped[LYRIC_KIND_MAIN].length
    ? grouped[LYRIC_KIND_MAIN]
    : available

  return {
    main: pickLyricByLanguage(
      preferTimedLyrics(mainCandidates),
      preferredLanguage,
    ),
    translation: pickLyricByLanguage(
      preferTimedLyrics(grouped[LYRIC_KIND_TRANSLATION]),
      preferredLanguage,
    ),
    pronunciation: pickLyricByLanguage(
      preferTimedLyrics(grouped[LYRIC_KIND_PRONUNCIATION]),
      preferredLanguage,
    ),
  }
}

const buildBaseKaraokeLine = (line, index, offset = 0) => ({
  index,
  start: applyTimeOffset(line?.start, offset),
  end: applyTimeOffset(line?.end, offset),
  value: typeof line?.value === 'string' ? line.value : '',
  tokens: [],
  lanes: [],
})

const buildBaseKaraokeLines = (baseLines, offset = 0) =>
  baseLines.map((line, index) => buildBaseKaraokeLine(line, index, offset))

const minNullableTime = (...values) => {
  const times = values.filter((value) => value != null)
  return times.length > 0 ? Math.min(...times) : null
}

const maxNullableTime = (...values) => {
  const times = values.filter((value) => value != null)
  return times.length > 0 ? Math.max(...times) : null
}

const laneRoleRank = (line) => {
  const role = (line?.agentRole || line?.role || '').toLowerCase()
  switch (role) {
    case '':
    case 'main':
      return 0
    case 'voice':
      return 1
    case 'group':
    case 'chorus':
    case 'choir':
      return 2
    case 'bg':
    case 'background':
    case 'background vocals':
    case 'background-vocals':
    case 'backing':
    case 'backing vocals':
    case 'backing-vocals':
      return 3
    default:
      return 1
  }
}

const sortCueLineLanes = (lanes) =>
  [...lanes].sort((a, b) => {
    const byRole = laneRoleRank(a) - laneRoleRank(b)
    if (byRole !== 0) return byRole
    return a.order - b.order
  })

const buildLaneFromCueLine = (cueLine, laneIndex) => ({
  key: `${cueLine.index}-${cueLine.agentId || 'default'}-${laneIndex}`,
  index: cueLine.index,
  start: cueLine.start,
  end: cueLine.end,
  value: cueLine.value,
  role: cueLine.role,
  agentId: cueLine.agentId,
  agentName: cueLine.agentName,
  agentRole: cueLine.agentRole,
  tokens: cueLine.tokens,
})

const buildLineFromCueLineGroup = (index, group, baseLines, offset = 0) => {
  const baseLine = buildBaseKaraokeLine(baseLines[index] || {}, index, offset)
  const orderedGroup = sortCueLineLanes(group)
  const lanes = orderedGroup.map(buildLaneFromCueLine)
  const first = lanes[0] || {}
  const tokens = sortTokensByStart(lanes.flatMap((lane) => lane.tokens || []))
  const fallbackStart =
    tokens.find((token) => token.start != null)?.start ?? null
  const fallbackEnd =
    [...tokens].reverse().find((token) => token.end != null)?.end ?? null
  const laneStart = minNullableTime(
    ...lanes.flatMap((lane) => [lane.start, lane.tokens?.[0]?.start]),
  )
  const laneEnd = maxNullableTime(
    ...lanes.flatMap((lane) => [
      lane.end,
      [...(lane.tokens || [])].reverse().find((token) => token.end != null)
        ?.end,
    ]),
  )
  const fallbackValue = tokens
    .map((token) => token.value)
    .filter(Boolean)
    .join(' ')
    .replace(/\s+/g, ' ')
    .trim()
  const value = baseLine.value || first.value || fallbackValue

  return {
    ...baseLine,
    index,
    start: minNullableTime(
      baseLine.start,
      first.start,
      laneStart,
      fallbackStart,
    ),
    end: maxNullableTime(baseLine.end, first.end, laneEnd, fallbackEnd),
    value,
    agentId: first.agentId,
    agentName: first.agentName,
    agentRole: first.agentRole,
    tokens,
    lanes,
  }
}

export const buildKaraokeLinesFromCueLines = (
  rawCueLines,
  baseLines,
  agentLookup,
  offset = 0,
) => {
  const normalizedCueLines = rawCueLines.map((cueLine, fallbackIndex) => {
    const normalized = normalizeCueLine(
      cueLine,
      fallbackIndex,
      agentLookup,
      offset,
    )
    return {
      ...normalized,
      order: fallbackIndex,
      tokens: normalized.tokens.map((token) => ({
        ...token,
        role: normalized.role,
        agentId: normalized.agentId,
        agentName: normalized.agentName,
        agentRole: normalized.agentRole,
      })),
    }
  })

  const byIndex = new Map()
  for (const cueLine of normalizedCueLines) {
    if (!byIndex.has(cueLine.index)) byIndex.set(cueLine.index, [])
    byIndex.get(cueLine.index).push(cueLine)
  }

  const indexes = new Set(baseLines.map((_line, index) => index))
  byIndex.forEach((_group, index) => indexes.add(index))

  return Array.from(indexes)
    .sort((a, b) => a - b)
    .map((index) => {
      const group = byIndex.get(index) || []
      if (group.length === 0) {
        return buildBaseKaraokeLine(baseLines[index] || {}, index, offset)
      }
      return buildLineFromCueLineGroup(index, group, baseLines, offset)
    })
}

export const buildKaraokeLines = (structuredLyric) => {
  if (!structuredLyric) return []

  const offset = toTime(structuredLyric.offset) ?? 0
  const agentLookup = buildAgentLookup(structuredLyric)
  const baseLines = Array.isArray(structuredLyric.line)
    ? structuredLyric.line
    : []
  const rawCueLines = Array.isArray(structuredLyric.cueLine)
    ? structuredLyric.cueLine
    : []

  const lines =
    rawCueLines.length > 0
      ? buildKaraokeLinesFromCueLines(
          rawCueLines,
          baseLines,
          agentLookup,
          offset,
        )
      : buildBaseKaraokeLines(baseLines, offset)

  const renderableLines = lines.filter(
    (line) => line.value || line.tokens.length > 0,
  )
  const hasUntimedLines = renderableLines.some((line) => line.start == null)
  const normalized = renderableLines.sort((a, b) => {
    if (hasUntimedLines) return a.index - b.index
    if (a.start == null && b.start == null) return a.index - b.index
    if (a.start == null) return 1
    if (b.start == null) return -1
    if (a.start !== b.start) return a.start - b.start
    return a.index - b.index
  })

  for (let i = 0; i < normalized.length; i += 1) {
    if (normalized[i].end == null) {
      const nextStart = normalized[i + 1]?.start
      if (nextStart != null) normalized[i].end = nextStart
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
  if (!token) return { start: null, end: null }

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
    return { start: estimatedStart, end: estimatedEnd }
  }

  const prevEnd = toTime(prevToken?.end) ?? toTime(prevToken?.start)
  let start = toTime(token.start)
  if (start == null) start = prevEnd ?? estimatedStart ?? lineStart

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

export const hasUsableKaraokeTiming = (lines) =>
  Array.isArray(lines) &&
  lines.some(
    (line) =>
      toTime(line?.start) != null ||
      (Array.isArray(line?.tokens) &&
        line.tokens.some(
          (token) => toTime(token?.start) != null || toTime(token?.end) != null,
        )),
  )

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

  if (mainStart == null) return -1
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
      if (Math.abs(start - mainStart) > maxDelta) continue
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
  return { index, line: index >= 0 ? layerLines[index] : null }
}

export const buildHighlightedMainLine = (line) => line

export const buildHighlightedAuxLine = (_referenceLine, auxiliaryLine) =>
  auxiliaryLine ?? null
