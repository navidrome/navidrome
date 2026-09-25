import React, { useCallback, useMemo } from 'react'
import {
  Datagrid,
  DateField,
  EditButton,
  Filter,
  NullableBooleanInput,
  NumberField,
  ReferenceInput,
  SearchInput,
  SelectInput,
  TextField,
  useUpdate,
  useNotify,
  useRecordContext,
  BulkDeleteButton,
  useDataProvider,
  useListContext,
  usePermissions,
} from 'react-admin'
import Switch from '@material-ui/core/Switch'
import { makeStyles } from '@material-ui/core/styles'
import { useMediaQuery } from '@material-ui/core'
import {
  ArtworkAvatar,
  DurationField,
  List,
  LoveButton,
  Writable,
  isWritable,
  useSelectedFields,
  useResourceRefresh,
} from '../common'
import FavoriteIcon from '@material-ui/icons/Favorite'
import config from '../config'
import PlaylistListActions from './PlaylistListActions'
import ChangePublicStatusButton from './ChangePublicStatusButton'
import ReactDragListView from 'react-drag-listview'

const useStyles = makeStyles((theme) => ({
  button: {
    color: theme.palette.type === 'dark' ? 'white' : undefined,
  },
}))

const PlaylistFilter = (props) => {
  const { permissions } = usePermissions()
  return (
    <Filter {...props} variant={'outlined'}>
      <SearchInput source="q" alwaysOn />
      {permissions === 'admin' && (
        <ReferenceInput
          source="owner_id"
          label={'resources.playlist.fields.ownerName'}
          reference="user"
          perPage={0}
          sort={{ field: 'name', order: 'ASC' }}
          alwaysOn
        >
          <SelectInput optionText="name" />
        </ReferenceInput>
      )}
      {config.enableFavourites && (
        <NullableBooleanInput
          source="starred"
          label={<FavoriteIcon fontSize={'small'} />}
        />
      )}
    </Filter>
  )
}

const TogglePublicInput = ({ resource, source }) => {
  const record = useRecordContext()
  const notify = useNotify()
  const [togglePublic] = useUpdate(
    resource,
    record.id,
    {
      ...record,
      public: !record.public,
    },
    {
      undoable: false,
      onFailure: (error) => {
        notify('ra.page.error', 'warning')
      },
    },
  )

  const handleClick = (e) => {
    togglePublic()
    e.stopPropagation()
  }

  return (
    <Switch
      checked={record[source]}
      onClick={handleClick}
      disabled={!isWritable(record.ownerId)}
    />
  )
}

const ToggleAutoImport = ({ resource, source }) => {
  const record = useRecordContext()
  const notify = useNotify()
  const [ToggleAutoImport] = useUpdate(
    resource,
    record.id,
    {
      ...record,
      sync: !record.sync,
    },
    {
      undoable: false,
      onFailure: (error) => {
        notify('ra.page.error', 'warning')
      },
    },
  )
  const handleClick = (e) => {
    ToggleAutoImport()
    e.stopPropagation()
  }

  return record.path ? (
    <Switch
      checked={record[source]}
      onClick={handleClick}
      disabled={!isWritable(record.ownerId)}
    />
  ) : null
}

const PlaylistListBulkActions = (props) => {
  const classes = useStyles()
  return (
    <>
      <ChangePublicStatusButton
        public={true}
        {...props}
        className={classes.button}
      />
      <ChangePublicStatusButton
        public={false}
        {...props}
        className={classes.button}
      />
      <BulkDeleteButton {...props} className={classes.button} />
    </>
  )
}

// Datagrid reads `source`/`sortable`/`label` off this element for the column
// header; only record/resource are forwarded so they never leak onto the button.
export const PlaylistLove = ({ record, className }) => (
  <LoveButton record={record} resource={'playlist'} className={className} />
)
PlaylistLove.defaultProps = { source: 'starred', sortable: false }

const ReorderablePlaylistGrid = ({ children }) => {
  const { currentSort, filterValues, ids, refetch } = useListContext()
  const dataProvider = useDataProvider()
  const notify = useNotify()

  const handleDragEnd = useCallback(
    (from, to) => {
      if (from === to) return
      const orderedIDs = [...ids]
      orderedIDs.splice(to, 0, orderedIDs.splice(from, 1)[0])
      dataProvider
        .reorderPlaylists(orderedIDs)
        .then(refetch)
        .catch(() => notify('ra.page.error', 'warning'))
    },
    [dataProvider, ids, notify, refetch],
  )

  const isFiltered = Object.values(filterValues || {}).some(
    (value) => value !== undefined && value !== null && value !== '',
  )
  // Sorting and filtering are temporary views. Dragging is only meaningful
  // while displaying the complete saved order.
  if (currentSort?.field !== 'custom' || isFiltered) return children

  return (
    <ReactDragListView onDragEnd={handleDragEnd}>{children}</ReactDragListView>
  )
}

const PlaylistList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  useResourceRefresh('playlist')

  const toggleableFields = useMemo(
    () => ({
      ownerName: isDesktop && <TextField source="ownerName" />,
      songCount: !isXsmall && <NumberField source="songCount" />,
      duration: <DurationField source="duration" />,
      updatedAt: isDesktop && (
        <DateField source="updatedAt" sortByOrder={'DESC'} />
      ),
      public: !isXsmall && (
        <TogglePublicInput source="public" sortByOrder={'DESC'} />
      ),
      comment: <TextField source="comment" />,
      sync: !isXsmall && (
        <ToggleAutoImport source="sync" sortByOrder={'DESC'} />
      ),
      starred: config.enableFavourites && <PlaylistLove />,
    }),
    [isDesktop, isXsmall],
  )

  const columns = useSelectedFields({
    resource: 'playlist',
    columns: toggleableFields,
    defaultOff: ['comment'],
  })

  return (
    <List
      {...props}
      exporter={false}
      perPage={0}
      sort={{ field: 'custom', order: 'ASC' }}
      filters={<PlaylistFilter />}
      actions={<PlaylistListActions />}
      bulkActionButtons={!isXsmall && <PlaylistListBulkActions />}
    >
      <ReorderablePlaylistGrid>
        <Datagrid
          rowClick="show"
          isRowSelectable={(r) => isWritable(r?.ownerId)}
        >
          <ArtworkAvatar source="id" variant="square" />
          <TextField source="name" />
          {columns}
          <Writable>
            <EditButton />
          </Writable>
        </Datagrid>
      </ReorderablePlaylistGrid>
    </List>
  )
}

export default PlaylistList
