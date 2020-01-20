import React from 'react'
import {
  BooleanField,
  Datagrid,
  Filter,
  DateField,
  List,
  SearchInput,
  SimpleList,
  TextField
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'

const UserFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const UserList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))

  return (
    <List
      {...props}
      sort={{ field: 'name', order: 'ASC' }}
      exporter={false}
      filters={<UserFilter />}
    >
      {isXsmall ? (
        <SimpleList
          primaryText={(record) => record.name}
          tertiaryText={(record) => (record.isAdmin ? '[admin]️' : '')}
        />
      ) : (
        <Datagrid rowClick="edit">
          <TextField source="name" />
          <BooleanField source="isAdmin" />
          <DateField source="lastLoginAt" locales="pt-BR" />
          <DateField source="lastAccessAt" locales="pt-BR" />
          <DateField source="updatedAt" locales="pt-BR" />
        </Datagrid>
      )}
    </List>
  )
}

export default UserList
