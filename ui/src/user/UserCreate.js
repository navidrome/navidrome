import React from 'react'
import {
  BooleanInput,
  Create,
  TextInput,
  PasswordInput,
  required,
  email,
  SimpleForm,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'

const UserCreate = (props) => {
  const translate = useTranslate()
  const resourceName = translate('resources.user.name', { smart_count: 1 })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })
  return (
    <Create title={<Title subTitle={title} />} {...props}>
      <SimpleForm redirect="list">
        <TextInput source="userName" validate={[required()]} />
        <TextInput source="name" validate={[required()]} />
        <TextInput source="email" validate={[required(), email()]} />
        <PasswordInput source="password" validate={[required()]} />
        <BooleanInput source="isAdmin" defaultValue={false} />
      </SimpleForm>
    </Create>
  )
}

export default UserCreate
