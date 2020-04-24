import React from 'react'
import {
  Datagrid,
  Filter,
  List,
  NumberField,
  SearchInput,
  TextField,
} from 'react-admin'
import { Pagination, Title } from '../common'

const ArtistFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const artistRowClick = (id) => {
  const filter = { artist_id: id }
  return `/album?filter=${JSON.stringify(
    filter
  )}&order=ASC&sort=maxYear&displayedFilters={"compilation":true}`
}

const ArtistList = (props) => (
  <List
    {...props}
    title={<Title subTitle={'Artists'} />}
    sort={{ field: 'orderArtistName', order: 'ASC' }}
    exporter={false}
    bulkActionButtons={false}
    filters={<ArtistFilter />}
    perPage={15}
    pagination={<Pagination />}
  >
    <Datagrid rowClick={artistRowClick}>
      <TextField source="name" sortBy={'orderArtistName'} />
      <NumberField source="albumCount" />
    </Datagrid>
  </List>
)

export default ArtistList
