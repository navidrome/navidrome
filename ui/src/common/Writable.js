import { cloneElement, Children, isValidElement } from 'react'

export const isWritable = (ownerId) => {
  return (
    localStorage.getItem('userId') === ownerId ||
    localStorage.getItem('role') === 'admin'
  )
}

export const isReadOnly = (ownerId) => {
  return !isWritable(ownerId)
}

export const Writable = (props) => {
  const { record = {}, children } = props
  if (isWritable(record.ownerId)) {
    return Children.map(children, (child) =>
      isValidElement(child) ? cloneElement(child, props) : child,
    )
  }
  return null
}

export const isSmartPlaylist = (pls) => !!pls.rules

export const canChangeTracks = (pls) =>
  isWritable(pls.ownerId) && !isSmartPlaylist(pls)
