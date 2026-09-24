import {
  TextField,
  Edit,
  SimpleForm,
  useTranslate,
  DeleteButton,
  DeleteWithConfirmButton,
  SaveButton,
  Toolbar,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { Title } from '../common'
import PlayerApiKey from './PlayerApiKey'
import { playerInputs } from './playerInputs'

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

const PlayerEditToolbar = (props) => (
  <Toolbar {...props} classes={useToolbarStyles()}>
    <SaveButton />
    {props.record?.hasApiKey ? (
      <DeleteWithConfirmButton
        mutationMode="pessimistic"
        confirmTitle="resources.player.message.deleteWithKeyTitle"
        confirmContent="resources.player.message.deleteWithKeyContent"
      />
    ) : (
      <DeleteButton />
    )}
  </Toolbar>
)

const PlayerEdit = (props) => (
  <Edit title={<PlayerTitle />} {...props}>
    <SimpleForm variant={'outlined'} toolbar={<PlayerEditToolbar />}>
      {playerInputs()}
      <TextField source="client" />
      <TextField source="userName" />
      <PlayerApiKey />
    </SimpleForm>
  </Edit>
)

export default PlayerEdit
