import React from 'react'
import {
  Show,
  SimpleShowLayout,
  BooleanField,
  DateField,
  TextField
} from 'react-admin'
import { BitrateField } from './index'

const SongDetails = (props) => {
  return (
    <Show {...props} title=" ">
      <SimpleShowLayout>
        <TextField source="path" />
        <TextField source="albumArtist" />
        <TextField source="genre" />
        <BooleanField source="compilation" />
        <BitrateField source="bitRate" />
        <DateField source="updatedAt" showTime />
      </SimpleShowLayout>
    </Show>
  )
}

export default SongDetails
