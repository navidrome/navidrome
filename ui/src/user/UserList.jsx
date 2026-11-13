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
import { useDateLocale } from '../i18n/useDateLocale'
import { formatDateTime } from '../utils/formatters'

const UserFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput id="search" source="name" alwaysOn />
  </Filter>
)

const UserList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const locale = useDateLocale()

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
            record.lastLoginAt && formatDateTime(record.lastLoginAt, locale)
          }
          tertiaryText={(record) => (record.isAdmin ? '[admin]️' : '')}
        />
      ) : (
        <Datagrid rowClick="edit">
          <TextField source="userName" />
          <TextField source="name" />
          <BooleanField source="isAdmin" />
          <DateField source="lastLoginAt" sortByOrder={'DESC'} />
          <DateField source="lastAccessAt" sortByOrder={'DESC'} />
          <DateField source="updatedAt" sortByOrder={'DESC'} />
        </Datagrid>
      )}
    </List>
  )
}

export default UserList
