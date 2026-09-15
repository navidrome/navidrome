const EMPHASIS_ROLES = new Set([
  'adlib',
  'backing',
  'backing vocals',
  'backing-vocals',
  'background',
  'background vocals',
  'background-vocals',
  'bg',
  'choir',
  'chorus',
  'group',
  'harmony',
])

const normalizeRole = (role) =>
  String(role || '')
    .trim()
    .toLowerCase()

export const isEmphasisRole = (token) =>
  EMPHASIS_ROLES.has(normalizeRole(token?.role)) ||
  EMPHASIS_ROLES.has(normalizeRole(token?.agentRole))

export const buildEmphasisStyle = (token) =>
  isEmphasisRole(token) ? { fontStyle: 'italic' } : undefined

export const parseColorRGB = (color) => {
  const value = String(color || '').trim()
  const hex = value.match(/^#([0-9a-f]{3}|[0-9a-f]{6})$/i)
  if (hex) {
    const raw = hex[1]
    if (raw.length === 3) {
      return raw.split('').map((part) => parseInt(part + part, 16))
    }
    return [
      parseInt(raw.slice(0, 2), 16),
      parseInt(raw.slice(2, 4), 16),
      parseInt(raw.slice(4, 6), 16),
    ]
  }

  const rgb = value.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/)
  return rgb ? [parseInt(rgb[1]), parseInt(rgb[2]), parseInt(rgb[3])] : null
}
