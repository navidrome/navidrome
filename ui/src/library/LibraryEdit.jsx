import React, { useCallback } from 'react'
import {
  Edit,
  SimpleForm,
  TextInput,
  required,
  Toolbar,
  SaveButton,
  NumberField,
  DateField,
  FunctionField,
  useTranslate,
  useMutation,
  useNotify,
  useRedirect,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { Divider, Typography } from '@material-ui/core'
import DeleteLibraryButton from './DeleteLibraryButton'
import { Title } from '../common'
import { formatBytes } from '../utils/index.js'

const useStyles = makeStyles({
  toolbar: {
    display: 'flex',
    justifyContent: 'space-between',
  },
  divider: {
    marginTop: '1em',
    marginBottom: '1em',
  },
  stats: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr 1fr',
    gap: '0 1em',
    '& > *': {
      padding: '0.2em 0',
    },
  },
})

const LibraryTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.library.name', { smart_count: 1 })
  return (
    <Title subTitle={`${resourceName} ${record ? `"${record.name}"` : ''}`} />
  )
}

const LibraryToolbar = (props) => (
  <Toolbar {...props} classes={useStyles()}>
    <SaveButton />
    {props.record && props.record.id !== 1 && (
      <DeleteLibraryButton {...props} />
    )}
  </Toolbar>
)

const formatDuration = (totalSeconds) => {
  if (totalSeconds == null || totalSeconds < 0) {
    return '0s'
  }
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = Math.floor(totalSeconds % 60)

  const parts = []
  if (hours > 0) {
    parts.push(`${hours}h`)
  }
  if (minutes > 0) {
    parts.push(`${minutes}m`)
  }
  if (seconds > 0 || parts.length === 0) {
    parts.push(`${seconds}s`)
  }
  return parts.join(' ')
}

const LibraryEdit = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()
  const isFirstLibrary = props.id === '1'

  const save = useCallback(
    async (values) => {
      try {
        await mutate(
          {
            type: 'update',
            resource: 'library',
            payload: { id: props.id, data: values },
          },
          { returnPromise: true },
        )
        notify('resources.library.notifications.updated', 'info', {
          smart_count: 1,
        })
        redirect('/library')
      } catch (error) {
        if (error.body && error.body.errors) {
          return error.body.errors
        }
      }
    },
    [mutate, notify, redirect, props.id],
  )

  return (
    <Edit title={<LibraryTitle />} undoable={false} {...props}>
      <SimpleForm variant={'outlined'} toolbar={<LibraryToolbar />} save={save}>
        <TextInput source="name" validate={[required()]} />
        <TextInput
          source="path"
          validate={[required()]}
          fullWidth
          disabled={isFirstLibrary}
        />
        <Divider className={classes.divider} fullWidth />
        <Typography variant="h6" gutterBottom>
          Statistics
        </Typography>
        {/*eslint-disable-next-line react/no-unknown-property*/}
        <div className={classes.stats} fullWidth>
          <div>
            <Typography variant="caption" display="block" color="textSecondary">
              {translate('resources.library.fields.totalSongs')}
            </Typography>
            <NumberField source="totalSongs" />
          </div>
          <div>
            <Typography variant="caption" display="block" color="textSecondary">
              {translate('resources.library.fields.totalAlbums')}
            </Typography>
            <NumberField source="totalAlbums" />
          </div>
          <div>
            <Typography variant="caption" display="block" color="textSecondary">
              {translate('resources.library.fields.totalArtists')}
            </Typography>
            <NumberField source="totalArtists" />
          </div>
          <div>
            <Typography variant="caption" display="block" color="textSecondary">
              {translate('resources.library.fields.totalFolders')}
            </Typography>
            <NumberField source="totalFolders" />
          </div>
          <div>
            <Typography variant="caption" display="block" color="textSecondary">
              {translate('resources.library.fields.totalFiles')}
            </Typography>
            <NumberField source="totalFiles" />
          </div>
          <div>
            <Typography variant="caption" display="block" color="textSecondary">
              {translate('resources.library.fields.totalMissingFiles')}
            </Typography>
            <NumberField source="totalMissingFiles" />
          </div>
          <div>
            <Typography variant="caption" display="block" color="textSecondary">
              {translate('resources.library.fields.totalSize')}
            </Typography>
            <FunctionField render={(record) => formatBytes(record.totalSize)} />
          </div>
          <div>
            <Typography variant="caption" display="block" color="textSecondary">
              {translate('resources.library.fields.totalDuration')}
            </Typography>
            <FunctionField
              render={(record) => formatDuration(record.totalDuration)}
            />
          </div>
        </div>
        <Divider className={classes.divider} fullWidth />
        {/*eslint-disable-next-line react/no-unknown-property*/}
        <div className={classes.stats} fullWidth>
          <div>
            <Typography variant="caption" display="block" color="textSecondary">
              {translate('resources.library.fields.lastScanAt')}
            </Typography>
            <DateField source="lastScanAt" showTime />
          </div>
          <div>
            <Typography variant="caption" display="block" color="textSecondary">
              {translate('resources.library.fields.createdAt')}
            </Typography>
            <DateField source="createdAt" showTime />
          </div>
          <div>
            <Typography variant="caption" display="block" color="textSecondary">
              {translate('resources.library.fields.updatedAt')}
            </Typography>
            <DateField source="updatedAt" showTime />
          </div>
        </div>
      </SimpleForm>
    </Edit>
  )
}

export default LibraryEdit
