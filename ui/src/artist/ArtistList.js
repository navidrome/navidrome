import React from 'react'
import {
  Datagrid,
  Filter,
  NumberField,
  SearchInput,
  TextField,
} from 'react-admin'
import { artistLink, List } from '../common'

const ArtistFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const ArtistList = (props) => (
  <List
    {...props}
    sort={{ field: 'name', order: 'ASC' }}
    exporter={false}
    bulkActionButtons={false}
    filters={<ArtistFilter />}
  >
    <Datagrid rowClick={artistLink}>
      <TextField source="name" />
      <NumberField source="albumCount" />
      <NumberField source="songCount" />
    </Datagrid>
  </List>
)

export default ArtistList
