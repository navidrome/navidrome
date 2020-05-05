import React from 'react'
import { List, Datagrid, TextField, BooleanField, DateField } from 'react-admin'
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
      <TextField source="Name" />
      <TextField source="Owner" />
      <BooleanField source="Public" />
      <DateField source="UpdatedAt" />
      <DurationField source="Duration" />
    </Datagrid>
  </List>
)

export default PlaylistList
