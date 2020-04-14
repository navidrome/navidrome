import React from 'react'
import {
  Show,
  SimpleShowLayout,
  BooleanField,
  DateField,
  TextField
} from 'react-admin'
import { BitrateField, SizeField } from './index'

const SongDetails = (props) => {
  const { record } = props
  return (
    <Show {...props} title=" ">
      <SimpleShowLayout>
        <TextField source="path" />
        <TextField source="albumArtist" />
        <TextField source="genre" />
        <BooleanField source="compilation" />
        <BitrateField source="bitRate" />
        <DateField source="updatedAt" showTime />
        <SizeField source="size" />
        <TextField source="playCount" />
        {record.playCount > 0 && <DateField source="playDate" showTime />}
      </SimpleShowLayout>
    </Show>
  )
}

export default SongDetails
