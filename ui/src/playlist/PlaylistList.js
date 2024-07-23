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
} from 'react-admin'
import Switch from '@material-ui/core/Switch'
import { useMediaQuery } from '@material-ui/core'
import {
  DurationField,
  List,
  Writable,
  isWritable,
  useSelectedFields,
  useResourceRefresh,
} from '../common'
import PlaylistListActions from './PlaylistListActions'
import ChangePublicStatusButton from './ChangePublicStatusButton'

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
        console.log(error)
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
  console.log(record)
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
        console.log(error)
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

const PlaylistListBulkActions = (props) => (
  <>
    <ChangePublicStatusButton public={true} {...props} />
    <ChangePublicStatusButton public={false} {...props} />
    <BulkDeleteButton {...props} />
  </>
)

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
