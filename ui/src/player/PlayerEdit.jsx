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
  DeleteButton,
  SaveButton,
  Toolbar,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { Title } from '../common'
import config from '../config'
import { BITRATE_CHOICES } from '../consts'
import PlayerApiKey from './PlayerApiKey'

const PlayerTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.player.name', { smart_count: 1 })
  return <Title subTitle={`${resourceName} ${record ? record.name : ''}`} />
}

const useToolbarStyles = makeStyles({
  toolbar: {
    display: 'flex',
    justifyContent: 'space-between',
  },
})

// DeleteWithUndoButton leaks confirm props to the DOM, so only pass them when confirming.
const deleteWithKeyProps = {
  undoable: false,
  confirmTitle: 'resources.player.message.deleteWithKeyTitle',
  confirmContent: 'resources.player.message.deleteWithKeyContent',
}

const PlayerEditToolbar = (props) => (
  <Toolbar {...props} classes={useToolbarStyles()}>
    <SaveButton />
    <DeleteButton {...(props.record?.hasApiKey ? deleteWithKeyProps : {})} />
  </Toolbar>
)

const PlayerEdit = (props) => (
  <Edit title={<PlayerTitle />} {...props}>
    <SimpleForm variant={'outlined'} toolbar={<PlayerEditToolbar />}>
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
      <TextField source="client" />
      <TextField source="userName" />
      <PlayerApiKey />
    </SimpleForm>
  </Edit>
)

export default PlayerEdit
