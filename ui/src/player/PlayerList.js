import React from 'react'
import {
  Datagrid,
  TextField,
  DateField,
  FunctionField,
  ReferenceField,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { SimpleList, List } from '../common'

const PlayerList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  return (
    <List exporter={false} {...props}>
      {isXsmall ? (
        <SimpleList
          primaryText={(r) => r.client}
          secondaryText={(r) => r.userName}
          tertiaryText={(r) => (r.maxBitRate ? r.maxBitRate : '-')}
        />
      ) : (
        <Datagrid rowClick="edit">
          <TextField source="name" />
          <ReferenceField source="transcodingId" reference="transcoding">
            <TextField source="name" />
          </ReferenceField>
          <FunctionField
            source="maxBitRate"
            render={(r) => (r.maxBitRate ? r.maxBitRate : '-')}
          />
          <DateField source="lastSeen" showTime sortByOrder={'DESC'} />
        </Datagrid>
      )}
    </List>
  )
}

export default PlayerList
