import React from 'react'
import {
  Datagrid,
  DateField,
  Filter,
  List,
  SearchInput,
  TextField,
  useTranslate,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { SimpleList } from '../common'
import AddIcon from '@material-ui/icons/Add'
import { CreateButton } from 'react-admin'

const ApiKeyFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput id="search" source="name" alwaysOn />
  </Filter>
)

const ApiKeyList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const translate = useTranslate()
  return (
    <List
      {...props}
      actions={
        <CreateButton
          basePath="/apikey"
          icon={<AddIcon />}
          label={translate('resources.apikey.actions.add')}
        />
      }
      sort={{ field: 'createdAt', order: 'DESC' }}
      exporter={false}
      bulkActionButtons={false}
      filters={<ApiKeyFilter />}
    >
      {isXsmall ? (
        <SimpleList
          primaryText={(r) => r.name}
          secondaryText={(r) => r.key}
          tertiaryText={(r) => <DateField record={r} source="createdAt" />}
          linkType={'edit'}
        />
      ) : (
        <Datagrid rowClick="edit">
          <TextField source="name" />
          <TextField source="key" />
          <DateField source="createdAt" showTime />
        </Datagrid>
      )}
    </List>
  )
}

export default ApiKeyList
