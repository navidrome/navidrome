import React from 'react'
import {
  Edit,
  SimpleForm,
  TextInput,
  TextField,
  BooleanInput,
  required,
} from 'react-admin'

const PlaylistEdit = (props) => (
  <Edit {...props}>
    <SimpleForm redirect="list">
      <TextInput source="name" validate={required()} />
      <TextInput multiline source="comment" />
      <BooleanInput source="public" />
      <BooleanInput source="sync" />
      <TextField source="path" />
    </SimpleForm>
  </Edit>
)

export default PlaylistEdit
