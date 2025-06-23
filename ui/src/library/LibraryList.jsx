import React from 'react'
import {
  Datagrid,
  Filter,
  SearchInput,
  SimpleList,
  TextField,
  NumberField,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { List, DateField } from '../common'

const LibraryFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const LibraryList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))

  return (
    <List
      {...props}
      sort={{ field: 'name', order: 'ASC' }}
      exporter={false}
      bulkActionButtons={false}
      filters={<LibraryFilter />}
    >
      {isXsmall ? (
        <SimpleList
          primaryText={(record) => record.name}
          secondaryText={(record) => record.path}
        />
      ) : (
        <Datagrid rowClick="edit">
          <TextField source="id" />
          <TextField source="name" />
          <TextField source="path" />
          <NumberField source="totalSongs" label="Songs" />
          <NumberField source="totalAlbums" label="Albums" />
          <NumberField source="totalMissingFiles" label="Missing Files" />
          <DateField source="lastScanAt" label="Last Scan" sortByOrder={'DESC'} />
        </Datagrid>
      )}
    </List>
  )
}

export default LibraryList 