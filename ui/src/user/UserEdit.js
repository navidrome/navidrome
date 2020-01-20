import React from 'react'
import {
  TextInput,
  BooleanInput,
  DateField,
  PasswordInput,
  Edit,
  required,
  email,
  SimpleForm
} from 'react-admin'

const UserTitle = ({ record }) => {
  return <span>User {record ? record.name : ''}</span>
}
const UserEdit = (props) => (
  <Edit title={<UserTitle />} {...props}>
    <SimpleForm>
      <TextInput source="userName" validate={[required()]} />
      <TextInput source="name" validate={[required()]} />
      <TextInput source="email" validate={[required(), email()]} />
      <PasswordInput source="password" validate={[required()]} />
      <BooleanInput source="isAdmin" initialValue={false} />
      <DateField source="lastLoginAt" />
      <DateField source="lastAccessAt" />
      <DateField source="updatedAt" />
      <DateField source="createdAt" />
    </SimpleForm>
  </Edit>
)

export default UserEdit
