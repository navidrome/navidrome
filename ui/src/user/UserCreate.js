import React from 'react'
import {
  BooleanInput,
  Create,
  TextInput,
  PasswordInput,
  required,
  SimpleForm
} from 'react-admin'

const UserCreate = (props) => (
  <Create {...props}>
    <SimpleForm redirect="list">
      <TextInput source="name" validate={[required()]} />
      <PasswordInput source="password" validate={[required()]} />
      <BooleanInput source="isAdmin" initialValue={false} />
    </SimpleForm>
  </Create>
)

export default UserCreate
