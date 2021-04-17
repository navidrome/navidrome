import React from 'react'
import {
  Datagrid,
  DateField,
  EditButton,
  Filter,
  NumberField,
  SearchInput,
  TextField,
  useUpdate,
  useNotify,
} from 'react-admin'
import Switch from '@material-ui/core/Switch'
import { DurationField, List, Writable, isWritable } from '../common'
import useSelectedFields from '../common/useSelectedFields'
import PlaylistListActions from './PlaylistListActions'

const PlaylistFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const TogglePublicInput = ({ permissions, resource, record = {}, source }) => {
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

  const canChange =
    permissions === 'admin' ||
    localStorage.getItem('username') === record['owner']

  return (
    <Switch
      checked={record[source]}
      onClick={handleClick}
      disabled={!canChange}
    />
  )
}

const PlaylistList = ({ permissions, ...props }) => {
  const toggleableFields = {
    owner: <TextField source="owner" />,
    songCount: <NumberField source="songCount" />,
    duration: <DurationField source="duration" />,
    updatedAt: <DateField source="updatedAt" sortByOrder={'DESC'} />,
    public: (
      <TogglePublicInput
        source="public"
        permissions={permissions}
        sortByOrder={'DESC'}
      />
    ),
  }
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
      <Datagrid
        rowClick="show"
        isRowSelectable={(r) => isWritable(r && r.owner)}
      >
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
