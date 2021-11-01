import React, { useMemo } from 'react'
import {
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
} from 'react-admin'
import Switch from '@material-ui/core/Switch'
import { useMediaQuery } from '@material-ui/core'
import {
  DurationField,
  Writable,
  isWritable,
  useSelectedFields,
  useResourceRefresh,
  isSmartPlaylist,
} from '../common'
import PlaylistListActions from './PlaylistListActions'
import { List, Datagrid } from '../infiniteScroll'

const PlaylistFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput source="q" alwaysOn />
    <ReferenceInput
      source="owner_id"
      reference="user"
      perPage={0}
      sort={{ field: 'name', order: 'ASC' }}
      alwaysOn
    >
      <SelectInput optionText="name" />
    </ReferenceInput>
  </Filter>
)

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
    }
  )

  const handleClick = (e) => {
    togglePublic()
    e.stopPropagation()
  }

  return (
    <Switch
      checked={record[source]}
      onClick={handleClick}
      disabled={!isWritable(record.ownerId) || isSmartPlaylist(record)}
    />
  )
}

const PlaylistList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  useResourceRefresh('playlist')

  const toggleableFields = useMemo(
    () => ({
      ownerName: <TextField source="ownerName" />,
      songCount: isDesktop && <NumberField source="songCount" />,
      duration: isDesktop && <DurationField source="duration" />,
      updatedAt: isDesktop && (
        <DateField source="updatedAt" sortByOrder={'DESC'} />
      ),
      public: !isXsmall && (
        <TogglePublicInput source="public" sortByOrder={'DESC'} />
      ),
    }),
    [isDesktop, isXsmall]
  )

  const columns = useSelectedFields({
    resource: 'playlist',
    columns: toggleableFields,
  })

  return (
    <List
      {...props}
      exporter={false}
      filters={<PlaylistFilter />}
      actions={<PlaylistListActions />}
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
