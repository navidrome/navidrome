import React from 'react'
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
  changeLocaleSuccess,
} from 'react-admin'
import { Title } from '../common'
import { useHistory } from 'react-router-dom'
import { makeStyles } from '@material-ui/core/styles'

const useStyles = makeStyles({
  button: {
    backgroundColor: 'transparent',
    border: '1px solid white',
    borderRadius: 30,
    padding: ' 5px 10px',
    marginBottom: 20,
    color: 'white',
  },
})

const PlayerTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.player.name', { smart_count: 1 })
  return <Title subTitle={`${resourceName} ${record ? record.name : ''}`} />
}

const BackButton = () => {
  const history = useHistory()
  const classes = useStyles()
  return (
    <button className={classes.button} onClick={() => history.goBack()}>
      Back
    </button>
  )
}

const PlayerEdit = (props) => (
  <Edit title={<PlayerTitle />} {...props}>
    <SimpleForm variant={'outlined'}>
      <BackButton />
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
          { id: 0, name: '-' },
        ]}
      />
      <BooleanInput source="reportRealPath" fullWidth />
      <TextField source="client" />
      <TextField source="userName" />
    </SimpleForm>
  </Edit>
)

export default PlayerEdit
