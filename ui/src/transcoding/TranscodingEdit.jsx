import React from 'react'
import {
  Edit,
  required,
  SelectInput,
  SimpleForm,
  TextInput,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'
import { TranscodingNote } from './TranscodingNote'
import { BITRATE_CHOICES } from '../consts'

const TranscodingTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.transcoding.name', {
    smart_count: 1,
  })
  return <Title subTitle={`${resourceName} ${record ? record.name : ''}`} />
}

const TranscodingEdit = (props) => {
  return (
    <>
      <TranscodingNote message={'message.transcodingEnabled'} />

      <Edit title={<TranscodingTitle />} {...props}>
        <SimpleForm variant={'outlined'}>
          <TextInput source="name" validate={[required()]} />
          <TextInput source="targetFormat" validate={[required()]} />
          <SelectInput source="defaultBitRate" choices={BITRATE_CHOICES} />
          <TextInput source="command" fullWidth validate={[required()]} />
        </SimpleForm>
      </Edit>
    </>
  )
}

export default TranscodingEdit
