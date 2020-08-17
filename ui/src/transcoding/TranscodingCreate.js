import React from 'react'
import {
  TextInput,
  SelectInput,
  Create,
  required,
  SimpleForm,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'

const TranscodingTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.transcoding.name', {
    smart_count: 1,
  })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })
  return <Title subTitle={title} />
}

const TranscodingCreate = (props) => (
  <Create title={<TranscodingTitle />} {...props}>
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
        defaultValue={192}
      />
      <TextInput
        source="command"
        fullWidth
        validate={[required()]}
        helperText={
          <span>
            Substitutions: <br />
            %s: File path <br />
            %b: BitRate (in kbps)
            <br />
          </span>
        }
      />
    </SimpleForm>
  </Create>
)

export default TranscodingCreate
