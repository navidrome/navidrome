import { utf8ByteRangeToCodeUnitRange } from './lyrics'

export const buildSegmentsFromLine = (line) => {
  if (!line || !Array.isArray(line.tokens) || line.tokens.length === 0) {
    return [{ text: line?.value || '', token: null, tokenIndex: -1 }]
  }

  const text = line.value || ''
  const exactSegments = (() => {
    if (!text) return null

    const rangedTokens = line.tokens
      .map((token, tokenIndex) => ({
        token,
        tokenIndex,
        range: utf8ByteRangeToCodeUnitRange(
          text,
          token?.byteStart,
          token?.byteEnd,
        ),
      }))
      .filter((entry) => entry.range != null)

    if (
      rangedTokens.length !== line.tokens.length ||
      rangedTokens.length === 0
    ) {
      return null
    }

    rangedTokens.sort(
      (a, b) =>
        a.range.start - b.range.start ||
        a.range.end - b.range.end ||
        a.tokenIndex - b.tokenIndex,
    )

    const segments = []
    let cursor = 0
    for (const entry of rangedTokens) {
      if (entry.range.start < cursor) return null
      if (entry.range.start > cursor) {
        segments.push({
          text: text.slice(cursor, entry.range.start),
          token: null,
          tokenIndex: -1,
        })
      }
      segments.push({
        text: entry.range.text,
        token: entry.token,
        tokenIndex: entry.tokenIndex,
      })
      cursor = entry.range.end
    }

    if (cursor < text.length) {
      segments.push({ text: text.slice(cursor), token: null, tokenIndex: -1 })
    }

    return segments
  })()

  if (exactSegments) return exactSegments

  const matchedSegments = []
  const fallbackSegments = []
  let cursor = 0
  let allMatched = text.length > 0
  let anyMatched = false

  const pushFallbackSeparatorIfNeeded = (nextTokenText) => {
    if (fallbackSegments.length === 0) return
    const prevText = fallbackSegments[fallbackSegments.length - 1].text || ''
    if (!prevText || !nextTokenText) return
    if (/\s$/.test(prevText) || /^\s/.test(nextTokenText)) return
    if (/[A-Za-z0-9]$/.test(prevText) && /^[A-Za-z0-9]/.test(nextTokenText)) {
      fallbackSegments.push({ text: ' ', token: null, tokenIndex: -1 })
    }
  }

  for (let tokenIndex = 0; tokenIndex < line.tokens.length; tokenIndex += 1) {
    const token = line.tokens[tokenIndex]
    const tokenText = token.value || ''
    if (!tokenText) continue

    pushFallbackSeparatorIfNeeded(tokenText)
    fallbackSegments.push({ text: tokenText, token, tokenIndex })

    if (!text) {
      allMatched = false
      continue
    }

    const foundAt = text.indexOf(tokenText, cursor)
    const normalizedFoundAt =
      foundAt >= 0
        ? foundAt
        : text.toLowerCase().indexOf(tokenText.toLowerCase(), cursor)

    if (normalizedFoundAt >= 0) {
      anyMatched = true
      if (normalizedFoundAt > cursor) {
        matchedSegments.push({
          text: text.slice(cursor, normalizedFoundAt),
          token: null,
          tokenIndex: -1,
        })
      }
      const matchedTokenText = text.slice(
        normalizedFoundAt,
        normalizedFoundAt + tokenText.length,
      )
      matchedSegments.push({
        text: matchedTokenText || tokenText,
        token,
        tokenIndex,
      })
      cursor = normalizedFoundAt + tokenText.length
    } else {
      allMatched = false
    }
  }

  if (allMatched && anyMatched) {
    if (cursor < text.length) {
      matchedSegments.push({
        text: text.slice(cursor),
        token: null,
        tokenIndex: -1,
      })
    }
    return matchedSegments
  }

  if (fallbackSegments.length > 0) return fallbackSegments

  return [{ text, token: null, tokenIndex: -1 }]
}
