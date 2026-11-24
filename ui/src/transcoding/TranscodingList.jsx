import React from 'react'
import { Datagrid, SelectField, TextField } from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { SimpleList, List } from '../common'
import { TRANSCODING_BITRATE_CHOICES } from '../consts'
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
          tertiaryText={(r) => (
            <SelectField
              record={r}
              source="defaultBitRate"
              choices={TRANSCODING_BITRATE_CHOICES}
            />
          )}
        />
      ) : (
        <Datagrid rowClick={config.enableTranscodingConfig ? 'edit' : 'show'}>
          <TextField source="name" />
          <TextField source="targetFormat" />
          <SelectField
            source="defaultBitRate"
            choices={TRANSCODING_BITRATE_CHOICES}
          />
          <TextField source="command" />
        </Datagrid>
      )}
    </List>
  )
}

export default TranscodingList
