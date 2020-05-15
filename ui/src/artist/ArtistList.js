import React from 'react'
import {
  Datagrid,
  Filter,
  List,
  NumberField,
  SearchInput,
  TextField,
} from 'react-admin'
import { artistLink, Pagination, Title } from '../common'

const ArtistFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const ArtistList = (props) => (
  <List
    {...props}
    title={
      <Title subTitle={'resources.artist.name'} args={{ smart_count: 2 }} />
    }
    sort={{ field: 'name', order: 'ASC' }}
    exporter={false}
    bulkActionButtons={false}
    filters={<ArtistFilter />}
    perPage={15}
    pagination={<Pagination />}
  >
    <Datagrid rowClick={artistLink}>
      <TextField source="name" />
      <NumberField source="albumCount" />
      <NumberField source="songCount" />
    </Datagrid>
  </List>
)

export default ArtistList
