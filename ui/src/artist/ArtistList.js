import React from 'react'
import {
  Datagrid,
  Filter,
  NumberField,
  SearchInput,
  TextField,
} from 'react-admin'
import { List, useGetHandleArtistClick } from '../common'
import { withWidth } from '@material-ui/core'

const ArtistFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const ArtistList = ({ width, ...props }) => {
  const handleArtistLink = useGetHandleArtistClick(width)
  return (
    <List
      {...props}
      sort={{ field: 'name', order: 'ASC' }}
      exporter={false}
      bulkActionButtons={false}
      filters={<ArtistFilter />}
    >
      <Datagrid rowClick={handleArtistLink}>
        <TextField source="name" />
        <NumberField source="albumCount" sortByOrder={'DESC'} />
        <NumberField source="songCount" sortByOrder={'DESC'} />
      </Datagrid>
    </List>
  )
}

export default withWidth()(ArtistList)
