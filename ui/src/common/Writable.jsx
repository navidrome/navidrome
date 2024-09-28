import { Children, cloneElement, isValidElement } from 'react'
import { isWritable } from './playlistUtils.js'

export const Writable = (props) => {
  const { record = {}, children } = props
  if (isWritable(record.ownerId)) {
    return Children.map(children, (child) =>
      isValidElement(child) ? cloneElement(child, props) : child,
    )
  }
  return null
}
