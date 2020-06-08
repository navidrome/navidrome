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
import { DurationField, List } from '../common'
import Writable, { isWritable } from '../common/Writable'

const PlaylistFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const TogglePublicInput = ({ resource, record, source }) => {
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

  return <Switch checked={record[source]} onClick={handleClick} />
}

const PlaylistList = (props) => (
  <List {...props} exporter={false} filters={<PlaylistFilter />}>
    <Datagrid rowClick="show" isRowSelectable={(r) => isWritable(r && r.owner)}>
      <TextField source="name" />
      <TextField source="owner" />
      <NumberField source="songCount" />
      <DurationField source="duration" />
      <DateField source="updatedAt" />
      <TogglePublicInput source="public" />
      <Writable>
        <EditButton />
      </Writable>
      />
    </Datagrid>
  </List>
)

export default PlaylistList
