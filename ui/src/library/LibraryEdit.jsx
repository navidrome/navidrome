import React, { useCallback } from 'react'
import {
  Edit,
  FormWithRedirect,
  TextInput,
  required,
  SaveButton,
  DateField,
  useTranslate,
  useMutation,
  useNotify,
  useRedirect,
  NumberInput,
} from 'react-admin'
import { Typography, Box, Toolbar } from '@material-ui/core'
import DeleteLibraryButton from './DeleteLibraryButton'
import { Title } from '../common'
import { formatBytes, formatDuration2, formatNumber } from '../utils/index.js'

const LibraryTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.library.name', { smart_count: 1 })
  return (
    <Title subTitle={`${resourceName} ${record ? `\"${record.name}\"` : ''}`} />
  )
}

const CustomToolbar = (props) => (
  <Toolbar {...props}>
    <Box display="flex" justifyContent="space-between" width="100%">
      <SaveButton
        handleSubmitWithRedirect={props.handleSubmitWithRedirect}
        saving={props.saving}
        pristine={props.pristine}
      />
      <DeleteLibraryButton />
    </Box>
  </Toolbar>
)

const LibraryEdit = (props) => {
  const translate = useTranslate()
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()

  const save = useCallback(
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

  return (
    <Edit title={<LibraryTitle />} undoable={false} {...props}>
      <FormWithRedirect
        {...props}
        save={save}
        render={(formProps) => (
          <form onSubmit={formProps.handleSubmit}>
            <Box p="1em" maxWidth="800px">
              <Box display="flex">
                <Box flex={1} mr="1em">
                  {/* Basic Information */}
                  <Typography variant="h6" gutterBottom>
                    {translate('resources.library.sections.basic')}
                  </Typography>

                  <TextInput source="name" validate={[required()]} />
                  <TextInput source="path" validate={[required()]} fullWidth />

                  <Box mt="2em" />

                  {/* Statistics - Two Column Layout */}
                  <Typography variant="h6" gutterBottom>
                    {translate('resources.library.sections.statistics')}
                  </Typography>

                  <Box display="flex">
                    <Box flex={1} mr="0.5em">
                      <NumberInput
                        disabled
                        resource={'library'}
                        source={'totalSongs'}
                        fullWidth
                      />
                    </Box>
                    <Box flex={1} ml="0.5em">
                      <NumberInput
                        disabled
                        resource={'library'}
                        source={'totalAlbums'}
                        fullWidth
                      />
                    </Box>
                  </Box>

                  <Box display="flex">
                    <Box flex={1} mr="0.5em">
                      <NumberInput
                        disabled
                        resource={'library'}
                        source={'totalArtists'}
                        fullWidth
                      />
                    </Box>
                    <Box flex={1} ml="0.5em">
                      <TextInput
                        disabled
                        resource={'library'}
                        source={'totalSize'}
                        format={formatBytes}
                        fullWidth
                      />
                    </Box>
                  </Box>

                  <Box display="flex">
                    <Box flex={1} mr="0.5em">
                      <TextInput
                        disabled
                        resource={'library'}
                        source={'totalDuration'}
                        format={formatDuration2}
                        fullWidth
                      />
                    </Box>
                    <Box flex={1} ml="0.5em">
                      <TextInput
                        disabled
                        resource={'library'}
                        source={'totalMissingFiles'}
                        fullWidth
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
                      record={formProps.record}
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
                      record={formProps.record}
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
                      record={formProps.record}
                    />
                  </Box>
                </Box>
              </Box>
            </Box>

            <CustomToolbar
              handleSubmitWithRedirect={formProps.handleSubmitWithRedirect}
              pristine={formProps.pristine}
              saving={formProps.saving}
            />
          </form>
        )}
      />
    </Edit>
  )
}

export default LibraryEdit
