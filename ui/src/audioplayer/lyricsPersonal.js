import { formatSourceValue } from './lyricsSource'

// The enhanced lyrics response carries a `lyricsList.personal` node, while a
// preference mutation returns `lyricsPersonal`. Both are listener-scoped and
// must never be shared across accounts.
export const readLyricsListPersonal = (response) => {
  const personal = response?.json?.['subsonic-response']?.lyricsList?.personal
  return personal && typeof personal === 'object' ? personal : null
}

export const readMutationPersonal = (response) => {
  const personal = response?.json?.['subsonic-response']?.lyricsPersonal
  return personal && typeof personal === 'object' ? personal : null
}

const asList = (value) => (Array.isArray(value) ? value : value ? [value] : [])

export const personalOffsets = (personal) => asList(personal?.offset)

export const personalRejections = (personal) => asList(personal?.rejection)

export const personalSelectionVariantKey = (personal) => {
  const selection = personal?.selection
  if (!selection) return null
  if (selection.status !== 'pinned' && selection.status !== 'rebound') {
    return null
  }
  if (selection.selection) return `selection:${selection.selection}`
  if (selection.local?.token) return `local:${selection.local.token}`
  return null
}

export const personalOffsetFor = (personal, { selection, local } = {}) => {
  for (const offset of personalOffsets(personal)) {
    if (
      offset?.kind === 'local' &&
      local &&
      String(offset.local?.token || '') === String(local)
    ) {
      return Number(offset.offsetMs) || 0
    }
  }
  for (const offset of personalOffsets(personal)) {
    if (
      offset?.kind !== 'local' &&
      selection &&
      offset?.selection === selection
    ) {
      return Number(offset.offsetMs) || 0
    }
  }
  return 0
}

export const personalSelectionLabel = (personal, labels = {}) => {
  const selection = personal?.selection
  if (!selection) return ''
  if (selection.kind === 'local') {
    const kind =
      selection.local?.kind === 'embedded'
        ? labels.embeddedSource || 'Embedded lyrics'
        : labels.sidecarSource || 'Sidecar file'
    const format = formatSourceValue(selection.local?.format)
    const language = formatSourceValue(selection.local?.language)
    return [kind, format, language].filter(Boolean).join(' · ')
  }
  const provider = formatSourceValue(selection.provider)
  const format = formatSourceValue(selection.identity?.format)
  const language = formatSourceValue(selection.identity?.language)
  const role = formatSourceValue(selection.identity?.role)
  return [
    provider || labels.pluginSource || 'Lyrics source',
    format,
    language,
    role && role !== 'Main' ? role : '',
  ]
    .filter(Boolean)
    .join(' · ')
}

export const personalSelectionUnavailable = (personal) =>
  personal?.selection?.status === 'unavailable'
