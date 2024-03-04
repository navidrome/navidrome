import React from 'react'
import {
  BooleanField,
  Datagrid,
  Filter,
  SearchInput,
  SimpleList,
  TextField,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { List, DateField } from '../common'

const UserFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput id="search" source="name" alwaysOn />
  </Filter>
)

const UserList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))

  return (
    <List
      {...props}
      sort={{ field: 'userName', order: 'ASC' }}
      exporter={false}
      bulkActionButtons={false}
      filters={<UserFilter />}
    >
      {isXsmall ? (
        <SimpleList
          primaryText={(record) => record.userName}
          secondaryText={(record) =>
            record.lastLoginAt && new Date(record.lastLoginAt).toLocaleString()
          }
          tertiaryText={(record) => (record.isAdmin ? '[admin]️' : '')}
        />
      ) : (
        <Datagrid rowClick="edit">
          <TextField source="userName" />
          <TextField source="name" />
          <BooleanField source="isAdmin" />
          <BooleanField source="syncPlayqueue" />
          <DateField source="lastLoginAt" sortByOrder={'DESC'} />
          <DateField source="updatedAt" sortByOrder={'DESC'} />
        </Datagrid>
      )}
    </List>
  )
}

export default UserList
