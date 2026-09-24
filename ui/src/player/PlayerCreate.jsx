import React from 'react'
import {
  BooleanInput,
  Create,
  ReferenceInput,
  SelectInput,
  SimpleForm,
  TextInput,
  required,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'
import config from '../config'
import { BITRATE_CHOICES } from '../consts'

const PlayerCreateTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.player.name', { smart_count: 1 })
  return (
    <Title subTitle={translate('ra.page.create', { name: resourceName })} />
  )
}

const PlayerCreate = (props) => (
  <Create title={<PlayerCreateTitle />} {...props}>
    <SimpleForm variant="outlined" redirect="edit">
      <TextInput source="name" validate={[required()]} />
      <ReferenceInput
        source="transcodingId"
        reference="transcoding"
        sort={{ field: 'name', order: 'ASC' }}
      >
        <SelectInput source="name" resettable />
      </ReferenceInput>
      <SelectInput source="maxBitRate" resettable choices={BITRATE_CHOICES} />
      <BooleanInput source="reportRealPath" fullWidth />
      {(config.lastFMEnabled || config.listenBrainzEnabled) && (
        <BooleanInput source="scrobbleEnabled" fullWidth />
      )}
    </SimpleForm>
  </Create>
)

export default PlayerCreate
