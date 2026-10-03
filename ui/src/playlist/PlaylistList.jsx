import React, { useMemo } from 'react'
import {
  Datagrid,
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
  usePermissions,
} from 'react-admin'
import Switch from '@material-ui/core/Switch'
import { makeStyles } from '@material-ui/core/styles'
import { useMediaQuery } from '@material-ui/core'
import {
  DateField,
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

export const ToggleField = ({ resource, source }) => {
  const record = useRecordContext()
  const notify = useNotify()
  const [toggle] = useUpdate(
    resource,
    record?.id,
    {
      ...record,
      [source]: !record?.[source],
    },
    {
      undoable: false,
      onFailure: (error) => {
        notify('ra.page.error', 'warning')
      },
    },
  )

  const handleClick = (e) => {
    toggle()
    e.stopPropagation()
  }

  if (!record) return null

  return (
    <Switch
      checked={record[source]}
      onClick={handleClick}
      disabled={!isWritable(record.ownerId)}
    />
  )
}

export const ToggleAutoImport = (props) => {
  const record = useRecordContext()
  return record?.path ? <ToggleField {...props} /> : null
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
      public: !isXsmall && <ToggleField source="public" sortByOrder={'DESC'} />,
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
      sort={{ field: 'name', order: 'ASC' }}
      filters={<PlaylistFilter />}
      actions={<PlaylistListActions />}
      bulkActionButtons={!isXsmall && <PlaylistListBulkActions />}
    >
      <Datagrid rowClick="show" isRowSelectable={(r) => isWritable(r?.ownerId)}>
        <ArtworkAvatar source="id" variant="square" />
        <TextField source="name" />
        {columns}
        <Writable>
          <EditButton />
        </Writable>
      </Datagrid>
    </List>
  )
}

export default PlaylistList
