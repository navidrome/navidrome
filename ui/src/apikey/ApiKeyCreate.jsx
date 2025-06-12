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

const ApiKeyCreate = (props) => {
  const translate = useTranslate()
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()
  const resourceName = translate('resources.apikey.name', { smart_count: 1 })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })

  const save = useCallback(
    async (values) => {
      try {
        await mutate(
          {
            type: 'create',
            resource: 'apikey',
            payload: { data: values },
          },
          { returnPromise: true },
        )
        notify('resources.apikey.notifications.created', 'info', {
          smart_count: 1,
        })
        redirect('/apikey')
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
        <TextInput source="name" validate={[required()]} autoFocus />
      </SimpleForm>
    </Create>
  )
}

export default ApiKeyCreate
