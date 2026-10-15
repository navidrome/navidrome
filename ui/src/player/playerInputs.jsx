import React from 'react'
import {
  BooleanInput,
  ReferenceInput,
  SelectInput,
  TextInput,
  required,
} from 'react-admin'
import config from '../config'
import { BITRATE_CHOICES } from '../consts'

// Returned as an array, not a component, so SimpleForm still injects its props into each input.
export const playerInputs = () =>
  [
    <TextInput key="name" source="name" validate={[required()]} />,
    <ReferenceInput
      key="transcodingId"
      source="transcodingId"
      reference="transcoding"
      sort={{ field: 'name', order: 'ASC' }}
    >
      <SelectInput source="name" resettable />
    </ReferenceInput>,
    <SelectInput
      key="maxBitRate"
      source="maxBitRate"
      resettable
      choices={BITRATE_CHOICES}
    />,
    <BooleanInput key="reportRealPath" source="reportRealPath" fullWidth />,
    (config.lastFMEnabled || config.listenBrainzEnabled) && (
      <BooleanInput key="scrobbleEnabled" source="scrobbleEnabled" fullWidth />
    ),
  ].filter(Boolean)
