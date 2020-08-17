import React from 'react'
import {
  Create,
  SimpleForm,
  TextInput,
  BooleanInput,
  required,
} from 'react-admin'

const PlaylistCreate = (props) => (
  <Create {...props}>
    <SimpleForm redirect="list" variant={'outlined'}>
      <TextInput source="name" validate={required()} />
      <TextInput multiline source="comment" />
      <BooleanInput source="public" initialValue={true} />
    </SimpleForm>
  </Create>
)

export default PlaylistCreate
