import React from 'react'
import {
  BooleanField,
  Datagrid,
  DateField,
  Filter,
  List,
  NumberField,
  FunctionField,
  SearchInput,
  NumberInput,
  BooleanInput,
  Show,
  SimpleShowLayout,
  TextField
} from 'react-admin'
import { DurationField, Pagination, Title } from '../common'
import { useMediaQuery } from '@material-ui/core'

const AlbumFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="name" alwaysOn />
    <BooleanInput source="compilation" />
    <NumberInput source="year" />
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

const AlbumList = (props) => {
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  return (
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
      <Datagrid expand={<AlbumDetails />} rowClick={'show'}>
        <TextField source="name" />
        <FunctionField
          source="artist"
          render={(r) => (r.albumArtist ? r.albumArtist : r.artist)}
        />
        {isDesktop && <NumberField source="songCount" />}
        <TextField source="year" />
        {isDesktop && <DurationField source="duration" />}
      </Datagrid>
    </List>
  )
}
export default AlbumList
