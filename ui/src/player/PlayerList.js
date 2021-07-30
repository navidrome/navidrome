import React from 'react'
import {Datagrid, DateField, Filter, FunctionField, ReferenceField, SearchInput, TextField,} from 'react-admin'
import {useMediaQuery} from '@material-ui/core'
import {List, SimpleList} from '../common'

const PlayerFilter = (props) => (
    <Filter {...props} variant={'outlined'}>
      <SearchInput source="name" alwaysOn/>
    </Filter>
)

const UserField = () => (
    <ReferenceField source="userId" reference="user">
      <TextField source="userName"/>
    </ReferenceField>
)

const PlayerList = ({permissions, ...props}) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  return (
      <List
          {...props}
          sort={{field: 'lastSeen', order: 'DESC'}}
          exporter={false}
          filters={<PlayerFilter/>}
      >
        {isXsmall ? (
            <SimpleList
                primaryText={(r) => r.client}
                secondaryText={(r) => (r.maxBitRate ? r.maxBitRate : '-')}
            />
        ) : (
            <Datagrid rowClick="edit">
              <TextField source="name"/>
              {permissions === 'admin' && <UserField source="userName"/>}
              <ReferenceField source="transcodingId" reference="transcoding">
                <TextField source="name"/>
              </ReferenceField>
              <FunctionField
                  source="maxBitRate"
                  render={(r) => (r.maxBitRate ? r.maxBitRate : '-')}
              />
              <DateField source="lastSeen" showTime sortByOrder={'DESC'}/>
            </Datagrid>
        )}
      </List>
  )
}

export default PlayerList
