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
import { Title } from '../common'

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
      </SimpleForm>
    </Create>
  )
}

export default LibraryCreate
