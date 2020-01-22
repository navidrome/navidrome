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
import { DurationField, Title } from '../common'

const AlbumFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="name" alwaysOn />
    <TextInput source="artist" />
  </Filter>
)

const AlbumDetails = (props) => {
  return (
    <Show {...props} title=" ">
      <SimpleShowLayout>
        <TextField label="Album Artist" source="albumArtist" />
        <TextField source="genre" />
        <BooleanField source="compilation" />
        <DateField source="updatedAt" showTime />
      </SimpleShowLayout>
    </Show>
  )
}

const albumRowClick = (id, basePath, record) =>
  `/song?filter={"artist":"${record.artist}","album":"${record.name}","album_id":"${record.id}"}&order=ASC&sort=trackNumber`

const AlbumList = (props) => (
  <List
    {...props}
    title={<Title subTitle={'Albums'} />}
    sort={{ field: 'name', order: 'ASC' }}
    exporter={false}
    bulkActionButtons={false}
    filters={<AlbumFilter />}
    perPage={15}
  >
    <Datagrid expand={<AlbumDetails />} rowClick={albumRowClick}>
      <TextField source="name" />
      <TextField source="artist" />
      <NumberField source="songCount" />
      <TextField source="year" />
      <DurationField label="Time" source="duration" />
    </Datagrid>
  </List>
)

export default AlbumList
