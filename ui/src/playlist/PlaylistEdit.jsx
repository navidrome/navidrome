import {
  Edit,
  FormDataConsumer,
  SimpleForm,
  TextInput,
  BooleanInput,
  required,
  useTranslate,
  usePermissions,
  ReferenceInput,
  SelectInput,
} from 'react-admin'
import { isWritable, ReadOnlyTextField, Title } from '../common'

const SyncFragment = ({ formData, variant, ...rest }) => {
  if (!formData.path) return null
  return (
    <>
      <BooleanInput source="sync" {...rest} />
      <ReadOnlyTextField source="path" {...rest} />
    </>
  )
}

const PlaylistTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.playlist.name', { smart_count: 1 })
  return <Title subTitle={`${resourceName} "${record ? record.name : ''}"`} />
}

const PlaylistEditForm = (props) => {
  const { record } = props
  const { permissions } = usePermissions()
  return (
    <SimpleForm redirect="list" variant={'outlined'} {...props}>
      <TextInput source="name" validate={required()} />
      <TextInput
        multiline
        minRows={3}
        source="comment"
        fullWidth
        inputProps={{
          style: { resize: 'vertical' },
        }}
      />
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
        <ReadOnlyTextField source="ownerName" />
      )}
      <BooleanInput source="public" disabled={!isWritable(record.ownerId)} />
      <FormDataConsumer fullWidth>
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
