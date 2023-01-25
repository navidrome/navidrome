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
