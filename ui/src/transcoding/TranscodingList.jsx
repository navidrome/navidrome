import React from 'react'
import { Datagrid, TextField } from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { SimpleList, List } from '../common'
import config from '../config'

const TranscodingList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  return (
    <List
      {...props}
      exporter={false}
      bulkActionButtons={config.enableTranscodingConfig}
    >
      {isXsmall ? (
        <SimpleList
          primaryText={(r) => r.name}
          secondaryText={(r) => `format: ${r.targetFormat}`}
          tertiaryText={(r) => r.defaultBitRate}
        />
      ) : (
        <Datagrid rowClick={config.enableTranscodingConfig ? 'edit' : 'show'}>
          <TextField source="name" />
          <TextField source="targetFormat" />
          <TextField source="defaultBitRate" />
          <TextField source="command" />
        </Datagrid>
      )}
    </List>
  )
}

export default TranscodingList
