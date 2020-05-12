import React from 'react'
import {
  Datagrid,
  TextField,
  BooleanField,
  NumberField,
  DateField,
} from 'react-admin'
import { DurationField, List } from '../common'

const PlaylistList = (props) => (
  <List {...props} exporter={false}>
    <Datagrid rowClick="edit">
      <TextField source="name" />
      <TextField source="owner" />
      <BooleanField source="public" />
      <NumberField source="songCount" />
      <DurationField source="duration" />
      <DateField source="updatedAt" />
    </Datagrid>
  </List>
)

export default PlaylistList
