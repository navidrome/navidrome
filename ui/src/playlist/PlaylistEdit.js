import React, { Fragment } from 'react'

import {
  Edit,
  FormDataConsumer,
  SimpleForm,
  TextInput,
  TextField,
  BooleanInput,
  required,
} from 'react-admin'

const SyncFragment = ({ formData, ...rest }) => {
  return (
    <Fragment>
      {formData.path && <BooleanInput source="sync" {...rest} />}
      {formData.path && <TextField source="path" {...rest} />}
    </Fragment>
  )
}

const PlaylistEdit = (props) => (
  <Edit {...props}>
    <SimpleForm redirect="list">
      <TextInput source="name" validate={required()} />
      <TextInput multiline source="comment" />
      <BooleanInput source="public" />
      <FormDataConsumer>
        {(formDataProps) => <SyncFragment {...formDataProps} />}
      </FormDataConsumer>
    </SimpleForm>
  </Edit>
)

export default PlaylistEdit
