import React from 'react'
import { Datagrid, List, TextField } from 'react-admin'
import { Title } from '../common'

const TranscodingList = (props) => (
  <List title={<Title subTitle={'Transcodings'} />} exporter={false} {...props}>
    <Datagrid rowClick="edit">
      <TextField source="name" />
      <TextField source="targetFormat" />
      <TextField source="defaultBitRate" />
      <TextField source="command" />
    </Datagrid>
  </List>
)

export default TranscodingList
