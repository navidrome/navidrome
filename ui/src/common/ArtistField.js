import React from 'react'
import { TextField as RATextField } from 'react-admin'

export const ArtistField = (props) => {
  const arname = props.record.name
  return <RATextField title={arname} {...props} />
}
