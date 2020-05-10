import React from 'react'
import {
  List,
  Datagrid,
  TextField,
  BooleanField,
  NumberField,
  DateField,
} from 'react-admin'
import { DurationField, Title } from '../common'

const PlaylistList = (props) => (
  <List
    {...props}
    title={
      <Title subTitle={'resources.playlist.name'} args={{ smart_count: 2 }} />
    }
    exporter={false}
  >
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
