// Display labels for a lyrics source, shared by the controls and the
// download filename.

const UPPERCASE_SOURCE_VALUES = new Set([
  'elrc',
  'lrc',
  'srt',
  'ttml',
  'txt',
  'yaml',
])

export const formatSourceValue = (value) => {
  const normalized = typeof value === 'string' ? value.trim() : ''
  if (!normalized) return ''
  if (UPPERCASE_SOURCE_VALUES.has(normalized.toLowerCase())) {
    return normalized.toUpperCase()
  }
  if (normalized.toLowerCase() === 'plain') return 'Plain text'
  return normalized
    .replace(/[-_]+/g, ' ')
    .replace(/\b\w/g, (character) => character.toUpperCase())
}

export const getSourceName = (source, labels) => {
  switch (source?.type) {
    case 'embedded':
      return labels.embeddedSource || 'Embedded lyrics'
    case 'sidecar':
      return labels.sidecarSource || 'Sidecar file'
    case 'plugin':
      return source.name || labels.pluginSource || 'Lyrics plugin'
    default:
      return source?.name || formatSourceValue(source?.type) || 'Lyrics'
  }
}
