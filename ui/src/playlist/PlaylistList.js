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
  return (
    <List {...props} exporter={false} filters={<PlaylistFilter />}>
      <Datagrid
        rowClick="show"
        isRowSelectable={(r) => isWritable(r && r.owner)}
      >
        <TextField source="name" />
        <TextField source="owner" />
        <NumberField source="songCount" />
        <DurationField source="duration" />
        <DateField source="updatedAt" sortByOrder={'DESC'} />
        <TogglePublicInput
          source="public"
          permissions={permissions}
          sortByOrder={'DESC'}
        />
        <Writable>
          <EditButton />
        </Writable>
      </Datagrid>
    </List>
  )
}

export default PlaylistList
