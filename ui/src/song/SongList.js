import React from 'react'
import {
  BooleanField,
  Datagrid,
  DateField,
  Filter,
  List,
  NumberField,
  SearchInput,
  TextInput,
  Show,
  SimpleShowLayout,
  TextField
} from 'react-admin'
import { BitrateField, DurationField, Title } from '../common'

const SongFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="title" alwaysOn />
    <TextInput source="album" />
    <TextInput source="artist" />
  </Filter>
)

const SongDetails = (props) => {
  return (
    <Show {...props} title=" ">
      <SimpleShowLayout>
        <TextField source="path" />
        <TextField label="Album Artist" source="albumArtist" />
        <TextField source="genre" />
        <BooleanField source="compilation" />
        <BitrateField source="bitRate" />
        <DateField source="updatedAt" showTime />
      </SimpleShowLayout>
    </Show>
  )
}

const SongList = (props) => (
  <List
    {...props}
    title={<Title subTitle={'Songs'} />}
    sort={{ field: 'title', order: 'ASC' }}
    exporter={false}
    bulkActionButtons={false}
    filters={<SongFilter />}
    perPage={15}
  >
    <Datagrid expand={<SongDetails />}>
      <TextField source="title" />
      <TextField source="album" />
      <TextField source="artist" />
      <NumberField label="Track #" source="trackNumber" />
      <NumberField label="Disc #" source="discNumber" />
      <TextField source="year" />
      <DurationField label="Time" source="duration" />
    </Datagrid>
  </List>
)

export default SongList
