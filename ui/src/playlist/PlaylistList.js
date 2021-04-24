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
import { useMediaQuery } from '@material-ui/core'

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
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))

  return (
    <List {...props} exporter={false} filters={<PlaylistFilter />}>
      <Datagrid
        rowClick="show"
        isRowSelectable={(r) => isWritable(r && r.owner)}
      >
        <TextField source="name" />
        <TextField source="owner" />
        {isDesktop && <NumberField source="songCount" />}
        {isDesktop && <DurationField source="duration" />}
        {isDesktop && <DateField source="updatedAt" sortByOrder={'DESC'} />}
        {!isXsmall && (
          <TogglePublicInput
            source="public"
            permissions={permissions}
            sortByOrder={'DESC'}
          />
        )}
        <Writable>
          <EditButton />
        </Writable>
      </Datagrid>
    </List>
  )
}

export default PlaylistList
