import {
  Edit,
  SimpleForm,
  useTranslate,
  DeleteButton,
  DeleteWithConfirmButton,
  SaveButton,
  Toolbar,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { ReadOnlyField, Title } from '../common'
import ApiKeyInput from './ApiKeyInput'
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
  <Edit title={<PlayerTitle />} mutationMode="pessimistic" {...props}>
    <SimpleForm variant={'outlined'} toolbar={<PlayerEditToolbar />}>
      {playerInputs()}
      <ReadOnlyField source="client" />
      <ReadOnlyField source="userName" />
      <ApiKeyInput source="apiKey" />
    </SimpleForm>
  </Edit>
)

export default PlayerEdit
