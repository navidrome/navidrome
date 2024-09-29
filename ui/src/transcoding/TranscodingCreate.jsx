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
import { BITRATE_CHOICES } from '../consts'

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
    <SimpleForm redirect="list" variant={'outlined'}>
      <TextInput source="name" validate={[required()]} />
      <TextInput source="targetFormat" validate={[required()]} />
      <SelectInput
        source="defaultBitRate"
        choices={BITRATE_CHOICES}
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
