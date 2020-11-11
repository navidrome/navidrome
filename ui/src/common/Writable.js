import { cloneElement, Children, isValidElement } from 'react'

export const isWritable = (owner) => {
  return (
    localStorage.getItem('username') === owner ||
    localStorage.getItem('role') === 'admin'
  )
}

export const isReadOnly = (owner) => {
  return !isWritable(owner)
}

export const Writable = (props) => {
  const { record = {}, children } = props
  if (isWritable(record.owner)) {
    return Children.map(children, (child) =>
      isValidElement(child) ? cloneElement(child, props) : child
    )
  }
  return null
}
