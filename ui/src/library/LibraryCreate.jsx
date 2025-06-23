import React, { useCallback } from 'react'
import {
  Create,
  SimpleForm,
  TextInput,
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
        if (error.body.errors) {
          return error.body.errors
        }
      }
    },
    [mutate, notify, redirect],
  )

  return (
    <Create title={<Title subTitle={title} />} {...props}>
      <SimpleForm save={save} variant={'outlined'}>
        <TextInput source="name" validate={[required()]} />
        <TextInput source="path" validate={[required()]} />
      </SimpleForm>
    </Create>
  )
}

export default LibraryCreate 