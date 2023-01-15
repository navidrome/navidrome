import React, { useCallback } from 'react'
import {
  Create,
  required,
  SimpleForm,
  TextInput,
  useMutation,
  useNotify,
  useRedirect,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'

const RadioCreate = (props) => {
  const translate = useTranslate()
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()

  const resourceName = translate('resources.radio.name', { smart_count: 1 })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })

  const save = useCallback(
    async (values) => {
      try {
        await mutate(
          {
            type: 'create',
            resource: 'radio',
            payload: { data: values },
          },
          { returnPromise: true }
        )
        notify('resources.radio.notifications.created', 'info', {
          smart_count: 1,
        })
        redirect('/radio')
      } catch (error) {
        if (error.body.errors) {
          return error.body.errors
        }
      }
    },
    [mutate, notify, redirect]
  )

  return (
    <Create title={<Title subTitle={title} />} {...props}>
      <SimpleForm save={save} variant={'outlined'}>
        <TextInput source="name" validate={[required()]} />
        <TextInput
          type="url"
          source="streamUrl"
          fullWidth
          validate={[required()]}
        />
        <TextInput type="url" source="homepageUrl" fullWidth />
      </SimpleForm>
    </Create>
  )
}

export default RadioCreate
