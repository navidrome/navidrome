import React, { Fragment } from 'react'
import {
  BooleanField,
  Datagrid,
  DateField,
  Filter,
  List,
  NumberField,
  SearchInput,
  Show,
  SimpleShowLayout,
  TextField,
  TextInput
} from 'react-admin'
import { BitrateField, DurationField, Pagination, Title } from '../common'
import AddToQueueButton from './AddToQueueButton'
import PlayButton from './PlayButton'

const SongFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="title" alwaysOn />
    <TextInput source="album" />
    <TextInput source="artist" />
  </Filter>
)

const SongBulkActionButtons = (props) => (
  <Fragment>
    <AddToQueueButton {...props} />
  </Fragment>
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

const SongList = (props) => {
  return (
    <List
      {...props}
      title={<Title subTitle={'Songs'} />}
      sort={{ field: 'title', order: 'ASC' }}
      exporter={false}
      bulkActionButtons={<SongBulkActionButtons />}
      filters={<SongFilter />}
      perPage={15}
      pagination={<Pagination />}
    >
      <Datagrid expand={<SongDetails />}>
        <PlayButton {...props} />
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
}

export default SongList
