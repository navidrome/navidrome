import React from 'react'
import {
  Datagrid,
  Filter,
  SearchInput,
  SimpleList,
  TextField,
  NumberField,
  BooleanField,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { List, DateField, useResourceRefresh, SizeField } from '../common'
import LibraryListBulkActions from './LibraryListBulkActions'
import LibraryListActions from './LibraryListActions'

const LibraryFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const LibraryList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  useResourceRefresh('library')

  return (
    <List
      {...props}
      sort={{ field: 'name', order: 'ASC' }}
      exporter={false}
      bulkActionButtons={!isXsmall && <LibraryListBulkActions />}
      filters={<LibraryFilter />}
      actions={<LibraryListActions />}
    >
      {isXsmall ? (
        <SimpleList
          primaryText={(record) => record.name}
          secondaryText={(record) => record.path}
        />
      ) : (
        <Datagrid rowClick="edit">
          <TextField source="name" />
          <TextField source="path" />
          <BooleanField source="defaultNewUsers" />
          <NumberField source="totalSongs" label="Songs" />
          <NumberField source="totalAlbums" label="Albums" />
          <NumberField source="totalMissingFiles" label="Missing Files" />
          <SizeField source="totalSize" />
          <DateField
            source="lastScanAt"
            label="Last Scan"
            sortByOrder={'DESC'}
          />
        </Datagrid>
      )}
    </List>
  )
}

export default LibraryList
