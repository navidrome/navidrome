export const PID_DEFAULT = ''
export const PID_FOLDER = 'folder'
export const PID_CUSTOM = '__custom__'

export const pidAlbumMode = (value) => {
  if (!value) return PID_DEFAULT
  if (value === PID_FOLDER) return PID_FOLDER
  return PID_CUSTOM
}

export const pidChanged = (record = {}, values = {}) =>
  (record.pidAlbum || '') !== (values.pidAlbum || '') ||
  (record.pidTrack || '') !== (values.pidTrack || '')

// react-final-form's default parse turns '' into undefined, which deletes the
// key from values and leaves the field permanently dirty against a '' initialValue.
export const parsePidField = (value) => value ?? ''
