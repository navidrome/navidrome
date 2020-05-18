import React from 'react'
import {
  Datagrid,
  TextField,
  BooleanField,
  NumberField,
  DateField,
  Filter,
  SearchInput,
  EditButton,
} from 'react-admin'
import { DurationField, List } from '../common'

const PlaylistFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const PlaylistList = (props) => (
  <List {...props} exporter={false} filters={<PlaylistFilter />}>
    <Datagrid rowClick="show">
      <TextField source="name" />
      <TextField source="owner" />
      <BooleanField source="public" />
      <NumberField source="songCount" />
      <DurationField source="duration" />
      <DateField source="updatedAt" />
      <EditButton />
    </Datagrid>
  </List>
)

export default PlaylistList
