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
          <SelectInput
            source="defaultBitRate"
            choices={[
              { id: 32, name: '32' },
              { id: 48, name: '48' },
              { id: 64, name: '64' },
              { id: 80, name: '80' },
              { id: 96, name: '96' },
              { id: 112, name: '112' },
              { id: 128, name: '128' },
              { id: 160, name: '160' },
              { id: 192, name: '192' },
              { id: 256, name: '256' },
              { id: 320, name: '320' },
            ]}
          />
          <TextInput source="command" fullWidth validate={[required()]} />
        </SimpleForm>
      </Edit>
    </>
  )
}

export default TranscodingEdit
