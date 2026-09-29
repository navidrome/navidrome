export const PID_GLOBAL = 'global'
export const PID_FOLDER = 'folder'
export const PID_CUSTOM = 'custom'

// Maps a stored PID override to the dropdown choice
export const pidModeFromValue = (value, allowFolder) => {
  const v = (value || '').trim()
  if (v === '') return PID_GLOBAL
  if (allowFolder && v === PID_FOLDER) return PID_FOLDER
  return PID_CUSTOM
}

// Returns the value to store for a dropdown choice. Custom keeps the current value
export const pidValueForMode = (mode, currentValue) => {
  switch (mode) {
    case PID_GLOBAL:
      return ''
    case PID_FOLDER:
      return PID_FOLDER
    default:
      return currentValue || ''
  }
}

// Reports whether the form values change the effective PID spec of the saved record. Like the
// server, it trims, treats empty as the global value and compares case-insensitively
export const pidConfigChanged = (values, record, globals) => {
  const effective = (value, field) =>
    ((value || '').trim() || globals?.[field] || '').toLowerCase()
  return ['pidAlbum', 'pidTrack'].some(
    (field) =>
      effective(values[field], field) !== effective(record?.[field], field),
  )
}
