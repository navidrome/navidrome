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
import { DurationField, Pagination, Title } from '../common'

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
        <TextField source="albumArtist" />
        <TextField source="genre" />
        <BooleanField source="compilation" />
        <DateField source="updatedAt" showTime />
      </SimpleShowLayout>
    </Show>
  )
}

const albumRowClick = (id, basePath, record) => {
  const filter = { album: record.name, album_id: id }
  if (!record.compilation) {
    filter.artist = record.artist
  }
  return `/song?filter=${JSON.stringify(filter)}&order=ASC&sort=trackNumber`
}

const AlbumList = (props) => (
  <List
    {...props}
    title={<Title subTitle={'Albums'} />}
    sort={{ field: 'name', order: 'ASC' }}
    exporter={false}
    bulkActionButtons={false}
    filters={<AlbumFilter />}
    perPage={15}
    pagination={<Pagination />}
  >
    <Datagrid expand={<AlbumDetails />} rowClick={albumRowClick}>
      <TextField source="name" />
      <TextField source="artist" />
      <NumberField source="songCount" />
      <TextField source="year" />
      <DurationField source="duration" />
    </Datagrid>
  </List>
)

export default AlbumList
