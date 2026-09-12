const normalizeLanguageTag = (language) => {
  const normalized = (language || '').trim().replace(/_/g, '-')
  if (!normalized) return ''

  try {
    return Intl.getCanonicalLocales(normalized)[0].toLowerCase()
  } catch {
    return normalized.toLowerCase()
  }
}

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

const hasTimedCueLines = (lyric) =>
  Array.isArray(lyric?.cueLine) &&
  lyric.cueLine.some(
    (cueLine) =>
      toTime(cueLine?.start) != null ||
      (Array.isArray(cueLine?.cue) &&
        cueLine.cue.some((cue) => toTime(cue?.start) != null)),
  )

const hasTimedLines = (lyric) =>
  Boolean(
    lyric &&
    lyric.synced &&
    ((Array.isArray(lyric.line) &&
      lyric.line.some((line) => toTime(line.start) != null)) ||
      hasTimedCueLines(lyric)),
  )

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

const lyricsInPreferredLanguage = (lyrics, preferredLanguage) => {
  if (!Array.isArray(lyrics) || lyrics.length === 0) return null

  const preferred = normalizeLanguageTag(preferredLanguage)
  const preferredBase = preferred.split('-')[0]

  const preferredMatches = lyrics.filter((lyric) =>
    languageMatch(normalizeLanguageTag(lyric.lang), preferred),
  )
  if (preferredMatches.length > 0) return preferredMatches

  const baseMatches = lyrics.filter((lyric) =>
    languageMatch(normalizeLanguageTag(lyric.lang), preferredBase),
  )
  if (baseMatches.length > 0) return baseMatches

  const englishMatches = lyrics.filter((lyric) =>
    languageMatch(normalizeLanguageTag(lyric.lang), 'en'),
  )
  return englishMatches.length > 0 ? englishMatches : lyrics
}

const pickPreferredLyric = (lyrics, preferredLanguage) => {
  const preferred = lyricsInPreferredLanguage(lyrics, preferredLanguage)
  return preferred ? preferTimedLyrics(preferred)[0] || null : null
}

export const hasCueTiming = (structuredLyric) =>
  hasTimedCueLines(structuredLyric)

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
  try {
    if (typeof window !== 'undefined' && window.localStorage) {
      const stored = window.localStorage.getItem('locale')
      if (stored) return stored
    }
  } catch {
    // Fall back to the browser language when storage access is restricted.
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
    main: pickPreferredLyric(mainCandidates, preferredLanguage),
    translation: pickPreferredLyric(
      grouped[LYRIC_KIND_TRANSLATION],
      preferredLanguage,
    ),
    pronunciation: pickPreferredLyric(
      grouped[LYRIC_KIND_PRONUNCIATION],
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

  return lines
    .map((line) => ({
      ...line,
      renderable: Boolean(line.value?.trim() || line.tokens.length > 0),
    }))
    .sort((a, b) => a.index - b.index)
}

export const hasUsableKaraokeTiming = (lines) =>
  Array.isArray(lines) &&
  lines.some(
    (line) =>
      line?.renderable !== false &&
      (toTime(line?.start) != null ||
        (Array.isArray(line?.tokens) &&
          line.tokens.some(
            (token) =>
              toTime(token?.start) != null || toTime(token?.end) != null,
          ))),
  )
