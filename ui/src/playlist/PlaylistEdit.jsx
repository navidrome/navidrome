import {
  Edit,
  FormDataConsumer,
  SimpleForm,
  TextInput,
  TextField,
  BooleanInput,
  required,
  useTranslate,
  usePermissions,
  ReferenceInput,
  SelectInput,
} from 'react-admin'
import { useForm } from 'react-final-form'
import Tooltip from '@material-ui/core/Tooltip'
import { makeStyles } from '@material-ui/core/styles'
import { isWritable, isSmartPlaylist, Title } from '../common'

const useStyles = makeStyles({
  tooltipWrapper: {
    display: 'inline-block',
  },
})

const SyncFragment = ({ formData, variant, ...rest }) => {
  return (
    <>
      {formData.path && <BooleanInput source="sync" {...rest} />}
      {formData.path && <TextField source="path" {...rest} />}
    </>
  )
}

const PlaylistTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.playlist.name', { smart_count: 1 })
  return <Title subTitle={`${resourceName} "${record ? record.name : ''}"`} />
}

const PublicInput = ({ record, formData }) => {
  const translate = useTranslate()
  const classes = useStyles()
  const isGlobal = isSmartPlaylist(record) && formData?.global
  const disabled = !isWritable(record.ownerId) || isGlobal

  const input = <BooleanInput source="public" disabled={disabled} />

  if (isGlobal) {
    return (
      <Tooltip
        title={translate(
          'resources.playlist.message.globalPlaylistPublicDisabled',
        )}
      >
        <div className={classes.tooltipWrapper}>{input}</div>
      </Tooltip>
    )
  }
  return input
}

const GlobalInput = ({ record }) => {
  const form = useForm()
  const handleChange = (value) => {
    if (value) {
      form.change('public', true)
    }
  }
  return (
    <BooleanInput
      source="global"
      disabled={!isWritable(record.ownerId)}
      onChange={handleChange}
    />
  )
}

const PlaylistEditForm = (props) => {
  const { record } = props
  const { permissions } = usePermissions()
  const isSmart = isSmartPlaylist(record)
  return (
    <SimpleForm redirect="list" variant={'outlined'} {...props}>
      <TextInput source="name" validate={required()} />
      <TextInput multiline source="comment" />
      {permissions === 'admin' ? (
        <ReferenceInput
          source="ownerId"
          reference="user"
          perPage={0}
          sort={{ field: 'name', order: 'ASC' }}
        >
          <SelectInput
            label={'resources.playlist.fields.ownerName'}
            optionText="userName"
          />
        </ReferenceInput>
      ) : (
        <TextField source="ownerName" />
      )}
      <FormDataConsumer>
        {({ formData }) => <PublicInput record={record} formData={formData} />}
      </FormDataConsumer>
      {isSmart && <GlobalInput record={record} />}
      <FormDataConsumer>
        {(formDataProps) => <SyncFragment {...formDataProps} />}
      </FormDataConsumer>
    </SimpleForm>
  )
}

const PlaylistEdit = (props) => (
  <Edit title={<PlaylistTitle />} actions={false} {...props}>
    <PlaylistEditForm {...props} />
  </Edit>
)

export default PlaylistEdit
