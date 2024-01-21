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
  useListContext,
  CreateButton,
  useTranslate,
  BooleanField,
} from 'react-admin'
import Switch from '@material-ui/core/Switch'
import { styled, Typography, useMediaQuery } from '@material-ui/core'
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
import { Inbox } from '@material-ui/icons'
import config from '../config'
import { ImportButton } from './ImportButton'

const PREFIX = 'RaEmpty'

export const EmptyClasses = {
  message: `${PREFIX}-message`,
  icon: `${PREFIX}-icon`,
  toolbar: `${PREFIX}-toolbar`,
}

const Root = styled('span', {
  name: PREFIX,
  overridesResolver: (props, styles) => styles.root,
})(({ theme }) => ({
  flex: 1,
  [`& .${EmptyClasses.message}`]: {
    textAlign: 'center',
    opacity: theme.palette.mode === 'light' ? 0.5 : 0.8,
    margin: '0 1em',
    color:
      theme.palette.mode === 'light' ? 'inherit' : theme.palette.text.primary,
  },

  [`& .${EmptyClasses.icon}`]: {
    width: '9em',
    height: '9em',
  },

  [`& .${EmptyClasses.toolbar}`]: {
    textAlign: 'center',
    marginTop: '1em',
  },
}))

const Empty = () => {
  const translate = useTranslate()
  const { resource } = useListContext()

  const resourceName = translate(`resources.${resource}.forcedCaseName`, {
    smart_count: 0,
    _: resource,
  })

  const emptyMessage = translate('ra.page.empty', { name: resourceName })
  const inviteMessage = translate('ra.page.invite')

  return (
    <Root>
      <div className={EmptyClasses.message}>
        <Inbox className={EmptyClasses.icon} />
        <Typography variant="h4" paragraph>
          {translate(`resources.${resource}.empty`, {
            _: emptyMessage,
          })}
        </Typography>
        <Typography variant="body1">
          {translate(`resources.${resource}.invite`, {
            _: inviteMessage,
          })}
        </Typography>
      </div>
      <div className={EmptyClasses.toolbar}>
        <CreateButton variant="contained" />{' '}
        {config.listenBrainzEnabled && <ImportButton />}
      </div>
      <div className={EmptyClasses.toolbar}></div>
    </Root>
  )
}

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
      external: <BooleanField source="externalId" looseValue />,
      externalSync: <BooleanField source="externalSync" />,
    }),
    [isDesktop, isXsmall],
  )

  const columns = useSelectedFields({
    resource: 'playlist',
    columns: toggleableFields,
    defaultOff: ['comment', 'external', 'externalSync'],
  })

  return (
    <List
      {...props}
      empty={<Empty />}
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
