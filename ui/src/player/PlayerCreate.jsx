import React from 'react'
import {
  TextInput,
  BooleanInput,
  Create,
  required,
  SimpleForm,
  SelectInput,
  ReferenceInput,
} from 'react-admin'
import { BITRATE_CHOICES } from '../consts'
import config from '../config.js'
import { Title } from '../common'
import { useTranslate } from 'react-admin'

const PlayerCreateTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.player.name', { smart_count: 1 })
  return <Title subTitle={`${translate('ra.action.create')} ${resourceName}`} />
}

const PlayerCreate = (props) => {
  return (
    <Create title={<PlayerCreateTitle />} {...props}>
      <SimpleForm variant="outlined">
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
}

export default PlayerCreate
