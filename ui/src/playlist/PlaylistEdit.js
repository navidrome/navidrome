import React, { Fragment } from 'react'

import {
  Edit,
  FormDataConsumer,
  SimpleForm,
  TextInput,
  TextField,
  BooleanInput,
  required,
  useTranslate,
} from 'react-admin'
import { isSmartPlaylist, isWritable, Title } from '../common'

const SyncFragment = ({ formData, variant, ...rest }) => {
  return (
    <Fragment>
      {formData.path && <BooleanInput source="sync" {...rest} />}
      {formData.path && <TextField source="path" {...rest} />}
    </Fragment>
  )
}

const PlaylistTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.playlist.name', { smart_count: 1 })
  return <Title subTitle={`${resourceName} "${record ? record.name : ''}"`} />
}

const PlaylistEditForm = (props) => {
  const { record } = props
  return (
    <SimpleForm redirect="list" variant={'outlined'} {...props}>
      <TextInput source="name" validate={required()} />
      <TextInput multiline source="comment" />
      <BooleanInput
        source="public"
        disabled={!isWritable(record.owner) || isSmartPlaylist(record)}
      />
      <FormDataConsumer>
        {(formDataProps) => <SyncFragment {...formDataProps} />}
      </FormDataConsumer>
    </SimpleForm>
  )
}

const PlaylistEdit = (props) => (
  <Edit title={<PlaylistTitle />} {...props}>
    <PlaylistEditForm {...props} />
  </Edit>
)

export default PlaylistEdit
