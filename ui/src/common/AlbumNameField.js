import React from 'react'
import { TextField as RATextField } from 'react-admin'

export const AlbumField = (props) => {
  const alname = props.record.name
  return <RATextField title={alname} {...props} />
}
