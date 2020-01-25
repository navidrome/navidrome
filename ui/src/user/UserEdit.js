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
import { Title } from '../common'

const UserTitle = ({ record }) => {
  return <Title subTitle={`User ${record ? record.name : ''}`} />
}
const UserEdit = (props) => (
  <Edit title={<UserTitle />} {...props}>
    <SimpleForm>
      <TextInput source="userName" validate={[required()]} />
      <TextInput source="name" validate={[required()]} />
      <TextInput source="email" validate={[required(), email()]} />
      <PasswordInput source="password" validate={[required()]} />
      <BooleanInput source="isAdmin" initialValue={false} />
      <DateField source="lastLoginAt" showTime />
      <DateField source="lastAccessAt" showTime />
      <DateField source="updatedAt" showTime />
      <DateField source="createdAt" showTime />
    </SimpleForm>
  </Edit>
)

export default UserEdit
