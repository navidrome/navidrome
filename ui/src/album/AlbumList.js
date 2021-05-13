import React from 'react'
import { useSelector } from 'react-redux'
import { Redirect, useLocation } from 'react-router-dom'
import {
  AutocompleteInput,
  Filter,
  NullableBooleanInput,
  NumberInput,
  ReferenceInput,
  SearchInput,
  Pagination,
  useTranslate,
} from 'react-admin'
import FavoriteIcon from '@material-ui/icons/Favorite'
import { withWidth } from '@material-ui/core'
import { List, QuickFilter, Title, useAlbumsPerPage } from '../common'
import AlbumListActions from './AlbumListActions'
import AlbumListView from './AlbumListView'
import AlbumGridView from './AlbumGridView'
import { AddToPlaylistDialog } from '../dialogs'
import albumLists, { defaultAlbumList } from './albumLists'
import config from '../config'

const AlbumFilter = (props) => {
  const translate = useTranslate()
  return (
    <Filter {...props} variant={'outlined'}>
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
      {config.enableFavourites && (
        <QuickFilter
          source="starred"
          label={<FavoriteIcon fontSize={'small'} />}
          defaultValue={true}
        />
      )}
    </Filter>
  )
}

const AlbumListTitle = ({ albumListType }) => {
  const translate = useTranslate()
  let title = translate('resources.album.name', { smart_count: 2 })
  if (albumListType) {
    let listTitle = translate(`resources.album.lists.${albumListType}`, {
      smart_count: 2,
    })
    title = `${title} - ${listTitle}`
  }
  return <Title subTitle={title} args={{ smart_count: 2 }} />
}

const AlbumList = (props) => {
  const { width } = props
  const albumView = useSelector((state) => state.albumView)
  const [perPage, perPageOptions] = useAlbumsPerPage(width)
  const location = useLocation()

  const albumListType = location.pathname
    .replace(/^\/album/, '')
    .replace(/^\//, '')

  // If it does not have filter/sort params (usually coming from Menu),
  // reload with correct filter/sort params
  if (!location.search) {
    const type =
      albumListType || localStorage.getItem('defaultView') || defaultAlbumList
    const listParams = albumLists[type]
    if (listParams) {
      return <Redirect to={`/album/${type}?${listParams.params}`} />
    }
  }

  return (
    <>
      <List
        {...props}
        exporter={false}
        bulkActionButtons={false}
        actions={<AlbumListActions />}
        filters={<AlbumFilter />}
        perPage={perPage}
        pagination={<Pagination rowsPerPageOptions={perPageOptions} />}
        title={<AlbumListTitle albumListType={albumListType} />}
      >
        {albumView.grid ? (
          <AlbumGridView albumListType={albumListType} {...props} />
        ) : (
          <AlbumListView {...props} />
        )}
      </List>
      <AddToPlaylistDialog />
    </>
  )
}

export default withWidth()(AlbumList)
