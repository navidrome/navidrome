import React, { useCallback } from 'react'
import {
  BooleanInput,
  Create,
  TextInput,
  PasswordInput,
  required,
  email,
  SimpleForm,
  useTranslate,
  useMutation,
  useNotify,
  useRedirect,
} from 'react-admin'
import { Title } from '../common'

const UserCreate = (props) => {
  const translate = useTranslate()
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()
  const resourceName = translate('resources.user.name', { smart_count: 1 })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })

  const save = useCallback(
    async (values) => {
      try {
        await mutate(
          {
            type: 'create',
            resource: 'user',
            payload: { data: values },
          },
          { returnPromise: true },
        )
        notify('resources.user.notifications.created', 'info', {
          smart_count: 1,
        })
        redirect('/user')
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
        <TextInput
          spellCheck={false}
          source="userName"
          validate={[required()]}
        />
        <TextInput source="name" validate={[required()]} />
        <TextInput spellCheck={false} source="email" validate={[email()]} />
        <PasswordInput
          spellCheck={false}
          source="password"
          validate={[required()]}
        />
        <BooleanInput source="isAdmin" defaultValue={false} />
        <BooleanInput source="syncPlayqueue" defaultValue={false} />
      </SimpleForm>
    </Create>
  )
}

export default UserCreate
