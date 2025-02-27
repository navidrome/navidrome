import { useSelector } from 'react-redux'
import { Redirect, useLocation } from 'react-router-dom'
import {
  AutocompleteArrayInput,
  AutocompleteInput,
  Filter,
  NullableBooleanInput,
  NumberInput,
  Pagination,
  ReferenceArrayInput,
  ReferenceInput,
  SearchInput,
  useRefresh,
  useTranslate,
  useVersion,
} from 'react-admin'
import FavoriteIcon from '@material-ui/icons/Favorite'
import { withWidth } from '@material-ui/core'
import {
  List,
  QuickFilter,
  Title,
  useAlbumsPerPage,
  useResourceRefresh,
  useSetToggleableFields,
} from '../common'
import AlbumListActions from './AlbumListActions'
import AlbumTableView from './AlbumTableView'
import AlbumGridView from './AlbumGridView'
import albumLists, { defaultAlbumList } from './albumLists'
import config from '../config'
import AlbumInfo from './AlbumInfo'
import ExpandInfoDialog from '../dialogs/ExpandInfoDialog'
import inflection from 'inflection'
import { makeStyles } from '@material-ui/core/styles'

const useStyles = makeStyles({
  chip: {
    margin: 0,
    height: '24px',
  },
})

const AlbumFilter = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  return (
    <Filter {...props} variant={'outlined'}>
      <SearchInput id="search" source="name" alwaysOn />
      <ReferenceInput
        label={translate('resources.album.fields.artist')}
        source="artist_id"
        reference="artist"
        sort={{ field: 'name', order: 'ASC' }}
        filterToQuery={(searchText) => ({ name: [searchText] })}
      >
        <AutocompleteInput emptyText="-- None --" />
      </ReferenceInput>
      <ReferenceArrayInput
        label={translate('resources.album.fields.genre')}
        source="genre_id"
        reference="genre"
        perPage={0}
        sort={{ field: 'name', order: 'ASC' }}
        filterToQuery={(searchText) => ({ name: [searchText] })}
      >
        <AutocompleteArrayInput emptyText="-- None --" classes={classes} />
      </ReferenceArrayInput>
      <ReferenceInput
        label={translate('resources.album.fields.recordLabel')}
        source="recordlabel"
        reference="tag"
        perPage={0}
        sort={{ field: 'tagValue', order: 'ASC' }}
        filter={{ tag_name: 'recordlabel' }}
        filterToQuery={(searchText) => ({
          tag_value: [searchText],
        })}
      >
        <AutocompleteInput emptyText="-- None --" optionText="tagValue" />
      </ReferenceInput>
      <ReferenceArrayInput
        label={translate('resources.album.fields.grouping')}
        source="grouping"
        reference="tag"
        perPage={0}
        sort={{ field: 'tagValue', order: 'ASC' }}
        filter={{ tag_name: 'grouping' }}
        filterToQuery={(searchText) => ({
          tag_value: [searchText],
        })}
      >
        <AutocompleteArrayInput
          emptyText="-- None --"
          classes={classes}
          optionText="tagValue"
        />
      </ReferenceArrayInput>
      <ReferenceArrayInput
        label={translate('resources.album.fields.mood')}
        source="mood"
        reference="tag"
        perPage={0}
        sort={{ field: 'tagValue', order: 'ASC' }}
        filter={{ tag_name: 'mood' }}
        filterToQuery={(searchText) => ({
          tag_value: [searchText],
        })}
      >
        <AutocompleteArrayInput
          emptyText="-- None --"
          classes={classes}
          optionText="tagValue"
        />
      </ReferenceArrayInput>
      <ReferenceInput
        label={translate('resources.album.fields.media')}
        source="media"
        reference="tag"
        perPage={0}
        sort={{ field: 'tagValue', order: 'ASC' }}
        filter={{ tag_name: 'media' }}
        filterToQuery={(searchText) => ({
          tag_value: [searchText],
        })}
      >
        <AutocompleteInput emptyText="-- None --" optionText="tagValue" />
      </ReferenceInput>
      <ReferenceInput
        label={translate('resources.album.fields.releaseType')}
        source="releasetype"
        reference="tag"
        perPage={0}
        sort={{ field: 'tagValue', order: 'ASC' }}
        filter={{ tag_name: 'releasetype' }}
        filterToQuery={(searchText) => ({
          tag_value: [searchText],
        })}
      >
        <AutocompleteInput
          emptyText="-- None --"
          optionText={(record) =>
            record?.tagValue
              ? inflection.humanize(record?.tagValue)
              : '-- None --'
          }
        />
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

const randomStartingSeed = Math.random().toString()

const AlbumList = (props) => {
  const { width } = props
  const albumView = useSelector((state) => state.albumView)
  const [perPage, perPageOptions] = useAlbumsPerPage(width)
  const location = useLocation()
  const version = useVersion()
  const refresh = useRefresh()
  useResourceRefresh('album')

  const seed = `${randomStartingSeed}-${version}`

  const albumListType = location.pathname
    .replace(/^\/album/, '')
    .replace(/^\//, '')

  // Workaround to force album columns to appear the first time.
  // See https://github.com/navidrome/navidrome/pull/923#issuecomment-833004842
  // TODO: Find a better solution
  useSetToggleableFields(
    'album',
    [
      'artist',
      'songCount',
      'playCount',
      'year',
      'duration',
      'rating',
      'size',
      'createdAt',
    ],
    ['createdAt', 'size'],
  )

  // If it does not have filter/sort params (usually coming from Menu),
  // reload with correct filter/sort params
  if (!location.search) {
    const type =
      albumListType || localStorage.getItem('defaultView') || defaultAlbumList
    const listParams = albumLists[type]
    if (type === 'random') {
      refresh()
    }
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
        filter={{ seed }}
        actions={<AlbumListActions />}
        filters={<AlbumFilter />}
        perPage={perPage}
        pagination={<Pagination rowsPerPageOptions={perPageOptions} />}
        title={<AlbumListTitle albumListType={albumListType} />}
      >
        {albumView.grid ? (
          <AlbumGridView albumListType={albumListType} {...props} />
        ) : (
          <AlbumTableView {...props} />
        )}
      </List>
      <ExpandInfoDialog content={<AlbumInfo />} />
    </>
  )
}

const AlbumListWithWidth = withWidth()(AlbumList)

export default AlbumListWithWidth
