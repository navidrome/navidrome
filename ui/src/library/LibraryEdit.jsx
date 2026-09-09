import React, { useCallback, useState } from 'react'
import {
  Edit,
  FormWithRedirect,
  TextInput,
  BooleanInput,
  required,
  SaveButton,
  DateField,
  useTranslate,
  useMutation,
  useNotify,
  useRedirect,
  Toolbar,
  Confirm,
} from 'react-admin'
import PropTypes from 'prop-types'
import { Typography, Box, Link } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import DeleteLibraryButton from './DeleteLibraryButton'
import PIDAlbumInput from './PIDAlbumInput'
import { pidChanged, parsePidField } from './pidUtils'
import { Title } from '../common'
import { formatBytes, formatDuration2, formatNumber } from '../utils/index.js'

const PID_DOCS_URL =
  'https://www.navidrome.org/docs/usage/configuration/persistent-ids/'

const useStyles = makeStyles({
  toolbar: {
    display: 'flex',
    justifyContent: 'space-between',
  },
})

const LibraryTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.library.name', { smart_count: 1 })
  return (
    <Title subTitle={`${resourceName} ${record ? `"${record.name}"` : ''}`} />
  )
}

const CustomToolbar = ({ showDelete, ...props }) => (
  <Toolbar {...props} classes={useStyles()}>
    <SaveButton disabled={props.pristine} />
    {showDelete && (
      <DeleteLibraryButton
        record={props.record}
        resource="library"
        basePath="/library"
      />
    )}
  </Toolbar>
)

// Must be <Edit>'s direct child: that's the only element react-admin injects
// the real fetched record into (LibraryEdit's own props never get it).
const LibraryEditForm = ({ canDelete, canEditPath, record, ...formProps }) => {
  const translate = useTranslate()
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()
  const [pendingValues, setPendingValues] = useState(null)

  const doSave = useCallback(
    async (values) => {
      try {
        await mutate(
          {
            type: 'update',
            resource: 'library',
            payload: { id: values.id, data: values },
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
    [mutate, notify, redirect],
  )

  // Changing the PID spec triggers a full scan server-side; confirm first.
  const handleSave = useCallback(
    (values) => {
      if (pidChanged(record, values)) {
        setPendingValues(values)
        return undefined
      }
      return doSave(values)
    },
    [doSave, record],
  )

  return (
    <FormWithRedirect
      {...formProps}
      record={record}
      save={handleSave}
      render={(formRenderProps) => (
        <form onSubmit={formRenderProps.handleSubmit}>
          <Box p="1em" maxWidth="800px">
            <Box display="flex">
              <Box flex={1} mr="1em">
                {/* Basic Information */}
                <Typography variant="h6" gutterBottom>
                  {translate('resources.library.sections.basic')}
                </Typography>

                <TextInput
                  source="name"
                  label={translate('resources.library.fields.name')}
                  validate={[required()]}
                  variant="outlined"
                />
                <TextInput
                  source="path"
                  label={translate('resources.library.fields.path')}
                  validate={[required()]}
                  fullWidth
                  variant="outlined"
                  InputProps={{ readOnly: !canEditPath }} // Disable editing path for library 1
                />
                <BooleanInput
                  source="defaultNewUsers"
                  label={translate('resources.library.fields.defaultNewUsers')}
                  variant="outlined"
                />

                <Box mt="2em" />

                <Typography variant="h6" gutterBottom>
                  {translate('resources.library.sections.persistentIds')}
                </Typography>
                <Typography variant="body2" gutterBottom>
                  <Link
                    href={PID_DOCS_URL}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {translate('resources.library.pid.docsLink')}
                  </Link>
                </Typography>

                <PIDAlbumInput />

                <TextInput
                  source="pidTrack"
                  label={translate('resources.library.fields.pidTrack')}
                  helperText={translate('resources.library.pid.trackHelp')}
                  parse={parsePidField}
                  fullWidth
                  variant="outlined"
                />

                <Box mt="2em" />

                {/* Statistics - Two Column Layout */}
                <Typography variant="h6" gutterBottom>
                  {translate('resources.library.sections.statistics')}
                </Typography>

                <Box display="flex">
                  <Box flex={1} mr="0.5em">
                    <TextInput
                      InputProps={{ readOnly: true }}
                      resource={'library'}
                      source={'totalSongs'}
                      label={translate('resources.library.fields.totalSongs')}
                      fullWidth
                      variant="outlined"
                    />
                  </Box>
                  <Box flex={1} ml="0.5em">
                    <TextInput
                      InputProps={{ readOnly: true }}
                      resource={'library'}
                      source={'totalAlbums'}
                      label={translate('resources.library.fields.totalAlbums')}
                      fullWidth
                      variant="outlined"
                    />
                  </Box>
                </Box>

                <Box display="flex">
                  <Box flex={1} mr="0.5em">
                    <TextInput
                      InputProps={{ readOnly: true }}
                      resource={'library'}
                      source={'totalArtists'}
                      label={translate('resources.library.fields.totalArtists')}
                      fullWidth
                      variant="outlined"
                    />
                  </Box>
                  <Box flex={1} ml="0.5em">
                    <TextInput
                      InputProps={{ readOnly: true }}
                      resource={'library'}
                      source={'totalSize'}
                      label={translate('resources.library.fields.totalSize')}
                      format={(v) => formatBytes(v, 2)}
                      fullWidth
                      variant="outlined"
                    />
                  </Box>
                </Box>

                <Box display="flex">
                  <Box flex={1} mr="0.5em">
                    <TextInput
                      InputProps={{ readOnly: true }}
                      resource={'library'}
                      source={'totalDuration'}
                      label={translate(
                        'resources.library.fields.totalDuration',
                      )}
                      format={formatDuration2}
                      fullWidth
                      variant="outlined"
                    />
                  </Box>
                  <Box flex={1} ml="0.5em">
                    <TextInput
                      InputProps={{ readOnly: true }}
                      resource={'library'}
                      source={'totalMissingFiles'}
                      label={translate(
                        'resources.library.fields.totalMissingFiles',
                      )}
                      fullWidth
                      variant="outlined"
                    />
                  </Box>
                </Box>

                {/* Timestamps Section */}
                <Box mb="1em">
                  <Typography
                    variant="body2"
                    color="textSecondary"
                    gutterBottom
                  >
                    {translate('resources.library.fields.lastScanAt')}
                  </Typography>
                  <DateField
                    variant="body1"
                    source="lastScanAt"
                    showTime
                    record={formRenderProps.record}
                  />
                </Box>

                <Box mb="1em">
                  <Typography
                    variant="body2"
                    color="textSecondary"
                    gutterBottom
                  >
                    {translate('resources.library.fields.updatedAt')}
                  </Typography>
                  <DateField
                    variant="body1"
                    source="updatedAt"
                    showTime
                    record={formRenderProps.record}
                  />
                </Box>

                <Box mb="2em">
                  <Typography
                    variant="body2"
                    color="textSecondary"
                    gutterBottom
                  >
                    {translate('resources.library.fields.createdAt')}
                  </Typography>
                  <DateField
                    variant="body1"
                    source="createdAt"
                    showTime
                    record={formRenderProps.record}
                  />
                </Box>
              </Box>
            </Box>
          </Box>

          <CustomToolbar
            handleSubmitWithRedirect={formRenderProps.handleSubmitWithRedirect}
            pristine={formRenderProps.pristine}
            saving={formRenderProps.saving}
            record={formRenderProps.record}
            showDelete={canDelete}
          />

          <Confirm
            isOpen={pendingValues !== null}
            title={translate('resources.library.pid.confirmTitle')}
            content={translate('resources.library.pid.confirmContent')}
            onConfirm={() => {
              const values = pendingValues
              setPendingValues(null)
              doSave(values)
            }}
            onClose={() => setPendingValues(null)}
          />
        </form>
      )}
    />
  )
}

LibraryEditForm.propTypes = {
  canDelete: PropTypes.bool,
  canEditPath: PropTypes.bool,
  record: PropTypes.object,
}

const LibraryEdit = (props) => {
  // Library ID 1 is protected (main library)
  const canDelete = props.id !== '1'
  const canEditPath = props.id !== '1'

  return (
    <Edit title={<LibraryTitle />} undoable={false} {...props}>
      <LibraryEditForm canDelete={canDelete} canEditPath={canEditPath} />
    </Edit>
  )
}

export default LibraryEdit
