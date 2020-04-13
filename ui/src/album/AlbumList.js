import React from 'react'
import { useSelector } from 'react-redux'
import {
  AutocompleteInput,
  Filter,
  List,
  NullableBooleanInput,
  NumberInput,
  ReferenceInput,
  SearchInput,
  Pagination,
  useTranslate
} from 'react-admin'
import { Title } from '../common'
import { withWidth } from '@material-ui/core'
import AlbumListActions from './AlbumListActions'
import AlbumListView from './AlbumListView'
import AlbumGridView from './AlbumGridView'
import { ALBUM_MODE_LIST } from './albumState'

const AlbumFilter = (props) => {
  const translate = useTranslate()
  return (
    <Filter {...props}>
      <SearchInput source="name" alwaysOn />
      <ReferenceInput
        label={translate('resources.album.fields.artist')}
        source="artist_id"
        reference="artist"
        sort={{ field: 'name', order: 'ASC' }}
        filterToQuery={(searchText) => ({ name: [searchText] })}
      >
        <AutocompleteInput emptyText="-- None --" />
      </ReferenceInput>
      <NullableBooleanInput source="compilation" />
      <NumberInput source="year" />
    </Filter>
  )
}

const getPerPage = (width) => {
  if (width === 'xs') return 12
  if (width === 'sm') return 12
  if (width === 'md') return 15
  if (width === 'lg') return 18
  return 21
}

const getPerPageOptions = (width) => {
  const options = [3, 6, 12]
  if (width === 'xs') return [12]
  if (width === 'sm') return [12]
  if (width === 'md') return options.map((v) => v * 5)
  if (width === 'lg') return options.map((v) => v * 6)
  return options.map((v) => v * 7)
}

const AlbumList = (props) => {
  const { width } = props
  const albumView = useSelector((state) => state.albumView)
  let sort
  if (albumView.mode === ALBUM_MODE_LIST) {
    sort = { field: 'name', order: 'ASC' }
  } else {
    sort = { field: 'created_at', order: 'DESC' }
  }
  return (
    <List
      {...props}
      title={<Title subTitle={'Albums'} />}
      exporter={false}
      bulkActionButtons={false}
      actions={<AlbumListActions />}
      sort={sort}
      filters={<AlbumFilter />}
      perPage={getPerPage(width)}
      pagination={<Pagination rowsPerPageOptions={getPerPageOptions(width)} />}
    >
      {albumView.mode === ALBUM_MODE_LIST ? (
        <AlbumListView {...props} />
      ) : (
        <AlbumGridView {...props} />
      )}
    </List>
  )
}

export default withWidth()(AlbumList)
