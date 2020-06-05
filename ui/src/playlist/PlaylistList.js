import React from 'react'
import {
  BooleanField,
  Datagrid,
  DateField,
  EditButton,
  Filter,
  NumberField,
  SearchInput,
  TextField,
} from 'react-admin'
import { DurationField, List } from '../common'
import Writable, { isWritable } from './Writable'

const PlaylistFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const PlaylistList = (props) => (
  <List {...props} exporter={false} filters={<PlaylistFilter />}>
    <Datagrid rowClick="show" isRowSelectable={(r) => isWritable(r.owner)}>
      <TextField source="name" />
      <TextField source="owner" />
      <BooleanField source="public" />
      <NumberField source="songCount" />
      <DurationField source="duration" />
      <DateField source="updatedAt" />
      <Writable>
        <EditButton />
      </Writable>
      />
    </Datagrid>
  </List>
)

export default PlaylistList
