import React from 'react'
import { useSelector } from 'react-redux'
import { useLocation } from 'react-router-dom'
import {
  AutocompleteInput,
  Filter,
  NullableBooleanInput,
  NumberInput,
  ReferenceInput,
  SearchInput,
  Pagination,
  useTranslate,
  useListParams,
} from 'react-admin'
import { List, useAlbumsPerPage } from '../common'
import { withWidth } from '@material-ui/core'
import AlbumListActions from './AlbumListActions'
import AlbumListView from './AlbumListView'
import AlbumGridView from './AlbumGridView'
import { ALBUM_MODE_LIST } from './albumState'
import AddToPlaylistDialog from '../dialogs/AddToPlaylistDialog'

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

const AlbumList = (props) => {
  const { width, resource } = props
  const albumView = useSelector((state) => state.albumView)
  const [perPage, perPageOptions] = useAlbumsPerPage(width)
  const location = useLocation()

  const [query] = useListParams({
    resource,
    location,
    perPage,
  })
  const isArtistView = !!(query.filter && query.filter.artist_id)

  return (
    <>
      <List
        {...props}
        exporter={false}
        bulkActionButtons={false}
        actions={<AlbumListActions />}
        filters={<AlbumFilter />}
        sort={{ field: 'name', order: 'ASC' }}
        perPage={perPage}
        pagination={<Pagination rowsPerPageOptions={perPageOptions} />}
      >
        {albumView.mode === ALBUM_MODE_LIST ? (
          <AlbumListView {...props} />
        ) : (
          <AlbumGridView isArtistView={isArtistView} {...props} />
        )}
      </List>
      <AddToPlaylistDialog />
    </>
  )
}

export default withWidth()(AlbumList)
