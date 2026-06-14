import {
  TextInput,
  BooleanInput,
  TextField,
  Edit,
  required,
  SimpleForm,
  SelectInput,
  ReferenceInput,
  useTranslate,
  useRecordContext,
} from 'react-admin'
import { Title } from '../common'
import config from '../config'
import { BITRATE_CHOICES } from '../consts'

const PlayerTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.player.name', { smart_count: 1 })
  return <Title subTitle={`${resourceName} ${record ? record.name : ''}`} />
}

export const TranscodingInput = (props) => {
  const translate = useTranslate()
  const record = useRecordContext(props)
  const isWebPlayer = record?.client === 'NavidromeUI'
  return (
    <ReferenceInput
      {...props}
      source="transcodingId"
      reference="transcoding"
      sort={{ field: 'name', order: 'ASC' }}
    >
      <SelectInput
        source="name"
        resettable
        helperText={
          isWebPlayer
            ? translate('resources.player.helperTexts.transcodingId')
            : undefined
        }
      />
    </ReferenceInput>
  )
}

const PlayerEdit = (props) => (
  <Edit title={<PlayerTitle />} {...props}>
    <SimpleForm variant={'outlined'}>
      <TextInput source="name" validate={[required()]} />
      <TranscodingInput />
      <SelectInput source="maxBitRate" resettable choices={BITRATE_CHOICES} />
      <BooleanInput source="reportRealPath" fullWidth />
      {(config.lastFMEnabled || config.listenBrainzEnabled) && (
        <BooleanInput source="scrobbleEnabled" fullWidth />
      )}
      <TextField source="client" />
      <TextField source="userName" />
    </SimpleForm>
  </Edit>
)

export default PlayerEdit
