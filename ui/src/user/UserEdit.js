import React from 'react'
import {
  TextInput,
  BooleanInput,
  DateField,
  PasswordInput,
  Edit,
  required,
  SimpleForm
} from 'react-admin'

const UserTitle = ({ record }) => {
  return <span>User {record ? record.name : ''}</span>
}
const UserEdit = (props) => (
  <Edit title={<UserTitle />} {...props}>
    <SimpleForm>
      <TextInput source="name" validate={[required()]} />
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
