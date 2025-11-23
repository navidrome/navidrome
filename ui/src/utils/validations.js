export const urlValidate = (value) => {
  if (!value) {
    return undefined
  }

  try {
    new URL(value)
    return undefined
  } catch (_) {
    return 'ra.validation.url'
  }
}

export function isDateSet(date) {
  if (!date) {
    return false
  }
  if (typeof date === 'string') {
    return date !== '0001-01-01T00:00:00Z'
  }
  if (date instanceof Date) {
    return date.toISOString() !== '0001-01-01T00:00:00Z'
  }
  return !!date
}
