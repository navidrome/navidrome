import React from 'react'
import {
  TextInput,
  TextField,
  Edit,
  required,
  SimpleForm,
  SelectInput,
  ReferenceInput,
} from 'react-admin'
import { Title } from '../common'

const PlayerTitle = ({ record }) => {
  return <Title subTitle={`Player ${record ? record.name : ''}`} />
}

const PlayerEdit = (props) => (
  <Edit title={<PlayerTitle />} {...props}>
    <SimpleForm>
      <TextInput source="name" validate={[required()]} />
      <ReferenceInput
        source="transcodingId"
        reference="transcoding"
        sort={{ field: 'name', order: 'ASC' }}
      >
        <SelectInput source="name" resettable />
      </ReferenceInput>
      <SelectInput
        source="maxBitRate"
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
          { id: 0, name: 'Unlimited' },
        ]}
      />
      <TextField source="client" />
      <TextField source="userName" />
    </SimpleForm>
  </Edit>
)

export default PlayerEdit
