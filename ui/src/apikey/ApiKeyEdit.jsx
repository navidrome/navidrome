import React, { useCallback } from 'react'
import {
  Edit,
  SimpleForm,
  TextInput,
  required,
  Toolbar,
  SaveButton,
  DeleteButton,
  useTranslate,
  useMutation,
  useNotify,
  useRefresh,
  DateField,
} from 'react-admin'
import { Title } from '../common'

const ApiKeyTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.apikey.name', { smart_count: 1 })
  return <Title subTitle={`${resourceName} ${record ? record.name : ''}`} />
}

const ApiKeyEditToolbar = (props) => (
  <Toolbar {...props}>
    <SaveButton />
    <DeleteButton />
  </Toolbar>
)

const ApiKeyEdit = (props) => {
  const [mutate] = useMutation()
  const notify = useNotify()
  const refresh = useRefresh()

  const save = useCallback(
    async (values) => {
      try {
        await mutate(
          {
            type: 'update',
            resource: 'apikey',
            payload: { id: values.id, data: values },
          },
          { returnPromise: true },
        )
        notify('resources.apikey.notifications.updated', 'info', {
          smart_count: 1,
        })
        refresh()
      } catch (error) {
        if (error.body.errors) {
          return error.body.errors
        }
      }
    },
    [mutate, notify, refresh],
  )

  return (
    <Edit title={<ApiKeyTitle />} {...props}>
      <SimpleForm
        toolbar={<ApiKeyEditToolbar />}
        save={save}
        variant={'outlined'}
      >
        <TextInput source="name" validate={[required()]} />
        <TextInput source="key" disabled fullWidth />
        <DateField variant="body1" source="createdAt" showTime />
      </SimpleForm>
    </Edit>
  )
}

export default ApiKeyEdit
