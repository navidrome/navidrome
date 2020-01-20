import React from 'react'
import {
  BooleanField,
  Datagrid,
  DateField,
  Filter,
  List,
  SearchInput,
  SimpleList,
  TextField
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'

const UserFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="q" alwaysOn />
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
          secondaryText={(record) => record.email}
        />
      ) : (
        <Datagrid>
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
