import React from 'react'
import {
  List,
  Datagrid,
  TextField,
  BooleanField,
  FunctionField,
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
      <FunctionField
        sortable={false} // TODO Make playlist.songCount sortable
        source="songCount"
        render={(r) => r.tracks && r.tracks.length}
      />
      <DurationField source="duration" />
      <DateField source="updatedAt" />
    </Datagrid>
  </List>
)

export default PlaylistList
