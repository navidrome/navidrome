export const isWritable = (ownerId) => {
  return (
    localStorage.getItem('userId') === ownerId ||
    localStorage.getItem('role') === 'admin'
  )
}

export const isReadOnly = (ownerId) => {
  return !isWritable(ownerId)
}

export const isSmartPlaylist = (pls) => !!pls.rules

export const isGlobalPlaylist = (pls) => isSmartPlaylist(pls) && pls.global

export const canChangeTracks = (pls) =>
  isWritable(pls.ownerId) && !isSmartPlaylist(pls)
