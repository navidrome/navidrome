import React from 'react'
import {
  Datagrid,
  List,
  TextField,
  DateField,
  FunctionField,
  ReferenceField
} from 'react-admin'
import { Title } from '../common'

const PlayerList = (props) => (
  <List title={<Title subTitle={'Players'} />} exporter={false} {...props}>
    <Datagrid rowClick="edit">
      <TextField source="name" />
      <ReferenceField
        label="Transcoding"
        source="transcodingId"
        reference="transcoding"
      >
        <TextField source="name" />
      </ReferenceField>
      <FunctionField
        label="MaxBitRate"
        source="maxBitRate"
        render={(r) => (r.maxBitRate ? r.maxBitRate : 'Unlimited')}
      />
      <DateField source="lastSeen" showTime />
    </Datagrid>
  </List>
)

export default PlayerList
