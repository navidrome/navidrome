import React, { useMemo } from 'react'
import {
  Datagrid,
  DateField,
  EditButton,
  Filter,
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
  useTranslate,
} from 'react-admin'
import Switch from '@material-ui/core/Switch'
import Tooltip from '@material-ui/core/Tooltip'
import { makeStyles } from '@material-ui/core/styles'
import { useMediaQuery } from '@material-ui/core'
import {
  DurationField,
  List,
  Writable,
  isWritable,
  isGlobalPlaylist,
  useSelectedFields,
  useResourceRefresh,
} from '../common'
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
    </Filter>
  )
}

const TogglePublicInput = ({ resource, source }) => {
  const record = useRecordContext()
  const notify = useNotify()
  const translate = useTranslate()
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

  const isGlobal = isGlobalPlaylist(record)
  const disabled = !isWritable(record.ownerId) || isGlobal

  const switchElement = (
    <Switch
      checked={record[source]}
      onClick={handleClick}
      disabled={disabled}
    />
  )

  if (isGlobal) {
    return (
      <Tooltip
        title={translate(
          'resources.playlist.message.globalPlaylistPublicDisabled',
        )}
      >
        <span>{switchElement}</span>
      </Tooltip>
    )
  }
  return switchElement
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
      sync: <ToggleAutoImport source="sync" sortByOrder={'DESC'} />,
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
