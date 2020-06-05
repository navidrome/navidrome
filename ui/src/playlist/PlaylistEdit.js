import React from 'react'
import {
  Edit,
  SimpleForm,
  TextInput,
  BooleanInput,
  required,
} from 'react-admin'

const PlaylistEdit = (props) => (
  <Edit {...props}>
    <SimpleForm redirect="list">
      <TextInput source="name" validate={required()} />
      <TextInput source="comment" />
      <BooleanInput source="public" />
    </SimpleForm>
  </Edit>
)

export default PlaylistEdit
