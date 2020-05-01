import React from 'react'
import {
  BooleanField,
  Datagrid,
  DateField,
  NumberField,
  FunctionField,
  Show,
  SimpleShowLayout,
  TextField,
} from 'react-admin'
import { DurationField, RangeField } from '../common'
import { useMediaQuery } from '@material-ui/core'
import AlbumContextMenu from './AlbumContextMenu'

const AlbumDetails = (props) => {
  return (
    <Show {...props} title=" ">
      <SimpleShowLayout>
        <TextField source="albumArtist" />
        <TextField source="genre" />
        <BooleanField source="compilation" />
        <DateField source="updatedAt" showTime />
      </SimpleShowLayout>
    </Show>
  )
}

const AlbumListView = ({ hasShow, hasEdit, hasList, ...rest }) => {
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  return (
    <Datagrid expand={<AlbumDetails />} rowClick={'show'} {...rest}>
      <TextField source="name" />
      <FunctionField
        source="artist"
        render={(r) => (r.albumArtist ? r.albumArtist : r.artist)}
      />
      {isDesktop && <NumberField source="songCount" />}
      {isDesktop && <NumberField source="playCount" />}
      {isDesktop && <RangeField source={'year'} sortBy={'maxYear'} />}
      {isDesktop && <DurationField source="duration" />}
      <AlbumContextMenu />
    </Datagrid>
  )
}
export default AlbumListView
