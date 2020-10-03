import React from 'react'
import {
  Create,
  SimpleForm,
  TextInput,
  BooleanInput,
  required,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'

const PlaylistCreate = (props) => {
  const translate = useTranslate()
  const resourceName = translate('resources.playlist.name', { smart_count: 1 })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })
  return (
    <Create title={<Title subTitle={title} />} {...props}>
      <SimpleForm redirect="list" variant={'outlined'}>
        <TextInput source="name" validate={required()} />
        <TextInput multiline source="comment" />
        <BooleanInput source="public" initialValue={true} />
      </SimpleForm>
    </Create>
  )
}

export default PlaylistCreate
