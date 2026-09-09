import React, { useCallback } from 'react'
import {
  Create,
  SimpleForm,
  TextInput,
  BooleanInput,
  required,
  useTranslate,
  useMutation,
  useNotify,
  useRedirect,
} from 'react-admin'
import { Typography, Box, Link } from '@material-ui/core'
import { Title } from '../common'
import PIDAlbumInput from './PIDAlbumInput'
import { parsePidField } from './pidUtils'

const PID_DOCS_URL =
  'https://www.navidrome.org/docs/usage/configuration/persistent-ids/'

const LibraryCreate = (props) => {
  const translate = useTranslate()
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()
  const resourceName = translate('resources.library.name', { smart_count: 1 })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })

  const save = useCallback(
    async (values) => {
      try {
        await mutate(
          {
            type: 'create',
            resource: 'library',
            payload: { data: values },
          },
          { returnPromise: true },
        )
        notify('resources.library.notifications.created', 'info', {
          smart_count: 1,
        })
        redirect('/library')
      } catch (error) {
        // Handle validation errors with proper field mapping
        if (error.body && error.body.errors) {
          return error.body.errors
        }

        // Handle other structured errors from the server
        if (error.body && error.body.error) {
          const errorMsg = error.body.error

          // Handle database constraint violations
          if (errorMsg.includes('UNIQUE constraint failed: library.name')) {
            return { name: 'ra.validation.unique' }
          }
          if (errorMsg.includes('UNIQUE constraint failed: library.path')) {
            return { path: 'ra.validation.unique' }
          }

          // Show a general notification for other server errors
          notify(errorMsg, 'error')
          return
        }

        // Fallback for unexpected error formats
        const fallbackMessage =
          error.message ||
          (typeof error === 'string' ? error : 'An unexpected error occurred')
        notify(fallbackMessage, 'error')
      }
    },
    [mutate, notify, redirect],
  )

  return (
    <Create title={<Title subTitle={title} />} {...props}>
      <SimpleForm save={save} variant={'outlined'}>
        <TextInput source="name" validate={[required()]} />
        <TextInput source="path" validate={[required()]} fullWidth />
        <BooleanInput source="defaultNewUsers" />

        <Box mt="1em" />

        <Typography variant="h6" gutterBottom>
          {translate('resources.library.sections.persistentIds')}
        </Typography>
        <Typography variant="body2" gutterBottom>
          <Link href={PID_DOCS_URL} target="_blank" rel="noopener noreferrer">
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
        />
      </SimpleForm>
    </Create>
  )
}

export default LibraryCreate
