import React from 'react'
import { TextField as RATextField } from 'react-admin'

export const SongArtistField = (props) => {
  const sarname = props.record.artist
  return <RATextField title={sarname} {...props} />
}
